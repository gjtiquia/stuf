package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"

	"stuf/internal/money"
	"stuf/internal/repo"
)

type OwedLedgerService struct {
	Store        *repo.Store
	Ledgers      *repo.OwedLedgerRepo
	Transactions *repo.OwedTransactionRepo
	Currency     *repo.CurrencyRepo
	History      HistoryService
	AppCurrency  string
}

type OwedTransactionService struct {
	Store        *repo.Store
	Ledgers      *repo.OwedLedgerRepo
	Transactions *repo.OwedTransactionRepo
	Currency     *repo.CurrencyRepo
	History      HistoryService
}

type OwedTransactionRow struct {
	Transaction repo.OwedTransaction
	Balance     money.Money
}

type owedLedgerMutationData struct {
	Ledger repo.OwedLedger
}

type owedTransactionMutationData struct {
	Transaction repo.OwedTransaction
}

func (s OwedLedgerService) Create(ctx context.Context, name, currencyCode, notes string) (repo.OwedLedger, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "owed ledger"); err != nil {
		return repo.OwedLedger{}, SessionEntry{}, err
	}
	if currencyCode == "" {
		currencyCode = s.AppCurrency
	}
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return repo.OwedLedger{}, SessionEntry{}, err
	}
	var out repo.OwedLedger
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		ledger, err := tx.Owed.Create(ctx, repo.OwedLedgerCreate{Name: name, CurrencyID: cur.ID, Notes: notes})
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "create", owedLedgerPath(ledger.Name), nil, owedLedgerMutationData{Ledger: ledger}, func(ctx context.Context) error {
			has, err := s.Ledgers.HasTransactions(ctx, ledger.ID)
			if err != nil {
				return err
			}
			if has {
				return nil
			}
			return s.Ledgers.Delete(ctx, ledger.ID)
		})
		if err != nil {
			return err
		}
		out, entry = ledger, e
		return nil
	})
	return out, entry, err
}

func (s OwedLedgerService) Update(ctx context.Context, id int64, name, currencyCode, notes string) (repo.OwedLedger, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "owed ledger"); err != nil {
		return repo.OwedLedger{}, SessionEntry{}, err
	}
	old, err := s.Ledgers.GetByID(ctx, id)
	if err != nil {
		return repo.OwedLedger{}, SessionEntry{}, err
	}
	currencyID := old.CurrencyID
	if currencyCode != "" && currencyCode != old.Code {
		cur, err := s.Currency.GetByCode(ctx, currencyCode)
		if err != nil {
			return repo.OwedLedger{}, SessionEntry{}, err
		}
		if err := s.validateLedgerCurrencyChange(ctx, old.ID, cur); err != nil {
			return repo.OwedLedger{}, SessionEntry{}, err
		}
		currencyID = cur.ID
	}
	next := old
	next.Name, next.CurrencyID, next.Notes = name, currencyID, notes
	var out repo.OwedLedger
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		updated, err := tx.Owed.Update(ctx, next)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", owedLedgerPath(updated.Name), owedLedgerMutationData{Ledger: old}, owedLedgerMutationData{Ledger: updated}, func(ctx context.Context) error {
			_, err := s.Ledgers.Update(ctx, old)
			return err
		})
		if err != nil {
			return err
		}
		out, entry = updated, e
		return nil
	})
	return out, entry, err
}

func (s OwedLedgerService) List(ctx context.Context) ([]repo.OwedLedger, error) {
	return s.Ledgers.List(ctx)
}

func (s OwedLedgerService) GetByName(ctx context.Context, name string) (repo.OwedLedger, error) {
	return s.Ledgers.GetByName(ctx, name)
}

func (s OwedLedgerService) GetByID(ctx context.Context, id int64) (repo.OwedLedger, error) {
	return s.Ledgers.GetByID(ctx, id)
}

func (s OwedLedgerService) validateLedgerCurrencyChange(ctx context.Context, ledgerID int64, next repo.Currency) error {
	txns, err := s.Transactions.ListByLedger(ctx, ledgerID)
	if err != nil {
		return err
	}
	for _, txn := range txns {
		cur, err := s.Currency.GetByID(ctx, txn.CurrencyID)
		if err != nil {
			return err
		}
		if _, err := convertMoney(txn.Amount, cur, next); err != nil {
			return fmt.Errorf("missing conversion for %s to %s", cur.Code, next.Code)
		}
	}
	return nil
}

func (s OwedTransactionService) Add(ctx context.Context, ledgerID int64, date, currencyCode, amountText, notes string) (repo.OwedTransaction, SessionEntry, error) {
	in, err := s.prepare(ctx, ledgerID, date, currencyCode, amountText, notes)
	if err != nil {
		return repo.OwedTransaction{}, SessionEntry{}, err
	}
	var out repo.OwedTransaction
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		txn, err := tx.OwedTx.Create(ctx, in)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "add", owedTransactionPath(txn), nil, owedTransactionMutationData{Transaction: txn}, func(ctx context.Context) error {
			return s.Transactions.Delete(ctx, txn.ID)
		})
		if err != nil {
			return err
		}
		out, entry = txn, e
		return nil
	})
	return out, entry, err
}

func (s OwedTransactionService) Update(ctx context.Context, id int64, date, currencyCode, amountText, notes string) (repo.OwedTransaction, SessionEntry, error) {
	old, err := s.Transactions.GetByID(ctx, id)
	if err != nil {
		return repo.OwedTransaction{}, SessionEntry{}, err
	}
	in, err := s.prepare(ctx, old.LedgerID, date, currencyCode, amountText, notes)
	if err != nil {
		return repo.OwedTransaction{}, SessionEntry{}, err
	}
	next := old
	next.Date, next.CurrencyID, next.Code, next.Amount, next.Formula, next.Notes = in.Date, in.CurrencyID, currencyCode, in.Amount, in.Formula, in.Notes
	var out repo.OwedTransaction
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		updated, err := tx.OwedTx.Update(ctx, next)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", owedTransactionPath(updated), owedTransactionMutationData{Transaction: old}, owedTransactionMutationData{Transaction: updated}, func(ctx context.Context) error {
			_, err := s.Transactions.Update(ctx, old)
			return err
		})
		if err != nil {
			return err
		}
		out, entry = updated, e
		return nil
	})
	return out, entry, err
}

func (s OwedTransactionService) Delete(ctx context.Context, id int64) (SessionEntry, error) {
	old, err := s.Transactions.GetByID(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		if err := tx.OwedTx.Delete(ctx, id); err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "delete", owedTransactionPath(old), owedTransactionMutationData{Transaction: old}, nil, func(ctx context.Context) error {
			_, err := s.Transactions.Create(ctx, repo.OwedTransactionCreate{LedgerID: old.LedgerID, Date: old.Date, CurrencyID: old.CurrencyID, Amount: old.Amount, Formula: old.Formula, Notes: old.Notes})
			return err
		})
		if err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

func (s OwedTransactionService) List(ctx context.Context, ledgerID int64) ([]repo.OwedTransaction, error) {
	return s.Transactions.ListByLedger(ctx, ledgerID)
}

func (s OwedTransactionService) GetByID(ctx context.Context, id int64) (repo.OwedTransaction, error) {
	return s.Transactions.GetByID(ctx, id)
}

func (s OwedTransactionService) GetByRef(ctx context.Context, id int64) (repo.OwedTransaction, error) {
	return s.Transactions.GetByID(ctx, id)
}

func (s OwedTransactionService) PreviewAmount(ctx context.Context, currencyCode, amountText string) (money.Money, string, error) {
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return money.Money{}, "", err
	}
	return parseOwedAmount(amountText, cur.Scale)
}

func (s OwedTransactionService) ListWithBalances(ctx context.Context, ledgerID int64) ([]OwedTransactionRow, error) {
	ledger, err := s.Ledgers.GetByID(ctx, ledgerID)
	if err != nil {
		return nil, err
	}
	txns, err := s.Transactions.ListByLedger(ctx, ledgerID)
	if err != nil {
		return nil, err
	}
	ledgerCur, err := s.Currency.GetByID(ctx, ledger.CurrencyID)
	if err != nil {
		return nil, err
	}
	out := make([]OwedTransactionRow, 0, len(txns))
	balance := money.Money{Scale: ledger.Scale}
	for _, txn := range txns {
		txnCur, err := s.Currency.GetByID(ctx, txn.CurrencyID)
		if err != nil {
			return nil, err
		}
		converted, err := convertMoney(txn.Amount, txnCur, ledgerCur)
		if err != nil {
			return nil, err
		}
		balance, err = balance.Add(converted)
		if err != nil {
			return nil, err
		}
		out = append(out, OwedTransactionRow{Transaction: txn, Balance: balance})
	}
	return out, nil
}

func (s OwedTransactionService) Balance(ctx context.Context, ledgerID int64) (money.Money, error) {
	rows, err := s.ListWithBalances(ctx, ledgerID)
	if err != nil {
		return money.Money{}, err
	}
	ledger, err := s.Ledgers.GetByID(ctx, ledgerID)
	if err != nil {
		return money.Money{}, err
	}
	if len(rows) == 0 {
		return money.Money{Scale: ledger.Scale}, nil
	}
	return rows[len(rows)-1].Balance, nil
}

func (s OwedTransactionService) NetTotal(ctx context.Context, appCurrency string) (money.Money, []string, error) {
	appCur, err := s.Currency.GetByCode(ctx, appCurrency)
	if err != nil {
		return money.Money{}, nil, err
	}
	ledgers, err := s.Ledgers.List(ctx)
	if err != nil {
		return money.Money{}, nil, err
	}
	total := money.Money{Scale: appCur.Scale}
	var warnings []string
	warned := map[string]bool{}
	for _, ledger := range ledgers {
		balance, err := s.Balance(ctx, ledger.ID)
		if err != nil {
			return money.Money{}, nil, err
		}
		ledgerCur, err := s.Currency.GetByID(ctx, ledger.CurrencyID)
		if err != nil {
			return money.Money{}, nil, err
		}
		converted, err := convertMoney(balance, ledgerCur, appCur)
		if err != nil {
			warning := fmt.Sprintf("missing conversion for %s", ledger.Code)
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
			continue
		}
		total, err = total.Add(converted)
		if err != nil {
			return money.Money{}, nil, err
		}
	}
	return total, warnings, nil
}

func (s OwedTransactionService) prepare(ctx context.Context, ledgerID int64, date, currencyCode, amountText, notes string) (repo.OwedTransactionCreate, error) {
	ledger, err := s.Ledgers.GetByID(ctx, ledgerID)
	if err != nil {
		return repo.OwedTransactionCreate{}, err
	}
	if !datePattern.MatchString(date) {
		return repo.OwedTransactionCreate{}, errors.New("date must be YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return repo.OwedTransactionCreate{}, errors.New("date must be a valid YYYY-MM-DD date")
	}
	if currencyCode == "" {
		currencyCode = ledger.Code
	}
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return repo.OwedTransactionCreate{}, err
	}
	amount, formula, err := parseOwedAmount(amountText, cur.Scale)
	if err != nil {
		return repo.OwedTransactionCreate{}, err
	}
	ledgerCur, err := s.Currency.GetByID(ctx, ledger.CurrencyID)
	if err != nil {
		return repo.OwedTransactionCreate{}, err
	}
	if _, err := convertMoney(amount, cur, ledgerCur); err != nil {
		return repo.OwedTransactionCreate{}, fmt.Errorf("missing conversion for %s to %s", cur.Code, ledger.Code)
	}
	return repo.OwedTransactionCreate{LedgerID: ledgerID, Date: date, CurrencyID: cur.ID, Amount: amount, Formula: formula, Notes: notes}, nil
}

func parseOwedAmount(input string, scale int) (money.Money, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return money.Money{}, "", errors.New("amount is required")
	}
	if strings.HasPrefix(input, "=") {
		amount, err := evalAmountFormula(strings.TrimSpace(strings.TrimPrefix(input, "=")), scale)
		if err != nil {
			return money.Money{}, "", err
		}
		return amount, input, nil
	}
	amount, err := money.NormalizeInput(input, scale)
	return amount, "", err
}

func evalAmountFormula(input string, scale int) (money.Money, error) {
	p := formulaParser{input: input}
	v, err := p.parseExpression()
	if err != nil {
		return money.Money{}, err
	}
	p.skipSpace()
	if p.pos != len(p.input) {
		return money.Money{}, errors.New("invalid formula")
	}
	mult := new(big.Rat).SetInt(big.NewInt(pow10(scale)))
	v.Mul(v, mult)
	n := roundRat(v)
	if !n.IsInt64() {
		return money.Money{}, errors.New("money amount overflow")
	}
	return money.Money{Amount: n.Int64(), Scale: scale}, nil
}

type formulaParser struct {
	input string
	pos   int
}

func (p *formulaParser) parseExpression() (*big.Rat, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if !p.consume('+') && !p.consume('-') {
			return left, nil
		}
		op := p.input[p.pos-1]
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		if op == '+' {
			left.Add(left, right)
		} else {
			left.Sub(left, right)
		}
	}
}

func (p *formulaParser) parseTerm() (*big.Rat, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if !p.consume('*') && !p.consume('/') {
			return left, nil
		}
		op := p.input[p.pos-1]
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		if op == '*' {
			left.Mul(left, right)
		} else {
			if right.Sign() == 0 {
				return nil, errors.New("formula cannot divide by zero")
			}
			left.Quo(left, right)
		}
	}
}

func (p *formulaParser) parseFactor() (*big.Rat, error) {
	p.skipSpace()
	if p.consume('+') {
		return p.parseFactor()
	}
	if p.consume('-') {
		v, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return v.Neg(v), nil
	}
	if p.consume('(') {
		v, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume(')') {
			return nil, errors.New("invalid formula")
		}
		return v, nil
	}
	return p.parseNumber()
}

func (p *formulaParser) parseNumber() (*big.Rat, error) {
	p.skipSpace()
	start := p.pos
	dot := false
	for p.pos < len(p.input) {
		r := rune(p.input[p.pos])
		if r == '.' && !dot {
			dot = true
			p.pos++
			continue
		}
		if !unicode.IsDigit(r) {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return nil, errors.New("invalid formula")
	}
	v := new(big.Rat)
	if _, ok := v.SetString(p.input[start:p.pos]); !ok {
		return nil, errors.New("invalid formula")
	}
	return v, nil
}

func (p *formulaParser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *formulaParser) consume(ch byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}

func roundRat(v *big.Rat) *big.Int {
	num := new(big.Int).Set(v.Num())
	den := new(big.Int).Set(v.Denom())
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(num, den, r)
	doubleR := new(big.Int).Abs(r)
	doubleR.Mul(doubleR, big.NewInt(2))
	if doubleR.Cmp(new(big.Int).Abs(den)) >= 0 {
		if num.Sign() == den.Sign() {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	return q
}

func convertMoney(amount money.Money, from, to repo.Currency) (money.Money, error) {
	if from.ID == to.ID || from.Code == to.Code {
		return amount.ConvertToScale(to.Scale)
	}
	return money.Convert(amount, from.RateToUSD, to.RateToUSD, to.Scale)
}

func pow10(scale int) int64 {
	var n int64 = 1
	for range scale {
		n *= 10
	}
	return n
}

func OwedTransactionRef(id int64) string {
	return fmt.Sprintf("txn-%06d", id)
}

func owedLedgerPath(name string) string {
	return "/owed/ledgers/" + name
}

func owedTransactionPath(t repo.OwedTransaction) string {
	return "/owed/ledgers/" + t.LedgerName + "/transactions/" + OwedTransactionRef(t.ID) + "/"
}
