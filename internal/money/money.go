package money

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Money struct {
	Amount int64
	Scale  int
}

var moneyPattern = regexp.MustCompile(`^\s*([+-]?)(\d+)(?:\.(\d+))?\s*$`)

func Parse(input string) (Money, error) {
	m := moneyPattern.FindStringSubmatch(input)
	if m == nil {
		return Money{}, fmt.Errorf("invalid money amount %q", input)
	}
	whole := m[2]
	frac := m[3]
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return Money{}, err
	}
	if m[1] == "-" {
		n = -n
	}
	return Money{Amount: n, Scale: len(frac)}, nil
}

func (m Money) Format(currencyCode string) string {
	sign := ""
	amount := m.Amount
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	if m.Scale == 0 {
		return fmt.Sprintf("%s %s%d", currencyCode, sign, amount)
	}
	div := pow10(m.Scale)
	return fmt.Sprintf("%s %s%d.%0*d", currencyCode, sign, amount/div, m.Scale, amount%div)
}

func (m Money) Add(other Money) (Money, error) {
	a, b, scale, err := align(m, other)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: a + b, Scale: scale}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	return m.Add(other.Negate())
}

func (m Money) Negate() Money {
	return Money{Amount: -m.Amount, Scale: m.Scale}
}

func (m Money) ConvertToScale(newScale int) (Money, error) {
	if newScale < 0 {
		return Money{}, errors.New("scale cannot be negative")
	}
	if newScale == m.Scale {
		return m, nil
	}
	if newScale > m.Scale {
		factor := pow10(newScale - m.Scale)
		if willMulOverflow(m.Amount, factor) {
			return Money{}, errors.New("money amount overflow")
		}
		return Money{Amount: m.Amount * factor, Scale: newScale}, nil
	}
	return Money{Amount: roundDiv(m.Amount, pow10(m.Scale-newScale)), Scale: newScale}, nil
}

func (m Money) Equals(other Money) bool {
	a, b, _, err := align(m, other)
	return err == nil && a == b
}

func (m Money) IsZero() bool     { return m.Amount == 0 }
func (m Money) IsPositive() bool { return m.Amount > 0 }
func (m Money) IsNegative() bool { return m.Amount < 0 }

func Convert(amount Money, rateToUSD Money, targetRateToUSD Money, targetScale int) (Money, error) {
	if rateToUSD.Amount <= 0 || targetRateToUSD.Amount <= 0 {
		return Money{}, errors.New("currency rates must be positive")
	}
	numerator := amount.Amount * rateToUSD.Amount * pow10(targetRateToUSD.Scale) * pow10(targetScale)
	denominator := pow10(amount.Scale) * pow10(rateToUSD.Scale) * targetRateToUSD.Amount
	return Money{Amount: roundDiv(numerator, denominator), Scale: targetScale}, nil
}

func align(a, b Money) (int64, int64, int, error) {
	if a.Scale == b.Scale {
		return a.Amount, b.Amount, a.Scale, nil
	}
	scale := max(a.Scale, b.Scale)
	aa, err := a.ConvertToScale(scale)
	if err != nil {
		return 0, 0, 0, err
	}
	bb, err := b.ConvertToScale(scale)
	if err != nil {
		return 0, 0, 0, err
	}
	return aa.Amount, bb.Amount, scale, nil
}

func pow10(scale int) int64 {
	var n int64 = 1
	for range scale {
		n *= 10
	}
	return n
}

func roundDiv(n, d int64) int64 {
	if d == 0 {
		panic("division by zero")
	}
	sign := int64(1)
	if n < 0 {
		sign = -1
		n = -n
	}
	q, r := n/d, n%d
	if r*2 >= d {
		q++
	}
	return sign * q
}

func willMulOverflow(a, b int64) bool {
	if a == 0 || b == 0 {
		return false
	}
	return math.Abs(float64(a)) > math.MaxInt64/math.Abs(float64(b))
}

func NormalizeInput(input string, scale int) (Money, error) {
	m, err := Parse(strings.TrimSpace(input))
	if err != nil {
		return Money{}, err
	}
	if m.Scale > scale {
		return Money{}, fmt.Errorf("amount has too many decimal places: max %d", scale)
	}
	return m.ConvertToScale(scale)
}
