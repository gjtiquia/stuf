package money

import "testing"

func TestParseFormatAndArithmetic(t *testing.T) {
	m, err := Parse("-123.45")
	if err != nil {
		t.Fatal(err)
	}
	if m.Amount != -12345 || m.Scale != 2 {
		t.Fatalf("unexpected parse: %+v", m)
	}
	if got := m.Format("HKD"); got != "HKD (123.45)" {
		t.Fatalf("format = %q", got)
	}
	sum, err := Money{Amount: 100, Scale: 1}.Add(Money{Amount: 25, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !sum.Equals(Money{Amount: 1025, Scale: 2}) {
		t.Fatalf("sum = %+v", sum)
	}
	diff, _ := sum.Sub(Money{Amount: 25, Scale: 2})
	if !diff.Equals(Money{Amount: 100, Scale: 1}) {
		t.Fatalf("diff = %+v", diff)
	}
}

func TestFormatDecimalText(t *testing.T) {
	tests := map[string]string{
		"":           "",
		"0":          "0",
		"123":        "123",
		"1234":       "1,234",
		"1234567":    "1,234,567",
		"1234.56":    "1,234.56",
		"-1234.56":   "-1,234.56",
		".5":         "0.5",
	}
	for input, want := range tests {
		if got := FormatDecimalText(input); got != want {
			t.Fatalf("FormatDecimalText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatWithThousandsSeparators(t *testing.T) {
	tests := []struct {
		m    Money
		code string
		want string
	}{
		{Money{Amount: 0, Scale: 2}, "HKD", "HKD 0.00"},
		{Money{Amount: -12345, Scale: 2}, "HKD", "HKD (123.45)"},
		{Money{Amount: 5000000, Scale: 2}, "HKD", "HKD 50,000.00"},
		{Money{Amount: 999900, Scale: 2}, "HKD", "HKD 9,999.00"},
		{Money{Amount: 1000000, Scale: 0}, "USD", "USD 1,000,000"},
	}
	for _, tc := range tests {
		if got := tc.m.Format(tc.code); got != tc.want {
			t.Fatalf("Format(%+v, %q) = %q, want %q", tc.m, tc.code, got, tc.want)
		}
	}
}

func TestRejectInvalidMoney(t *testing.T) {
	for _, input := range []string{"", ".", "1.2.3", "HKD 1", "1_000"} {
		if _, err := Parse(input); err == nil {
			t.Fatalf("expected %q to be invalid", input)
		}
	}
	if _, err := NormalizeInput("1.234", 2); err == nil {
		t.Fatal("expected too many decimal places to fail")
	}
}

func TestScaleConversionAndCurrencyConversion(t *testing.T) {
	m, err := (Money{Amount: 1235, Scale: 3}).ConvertToScale(2)
	if err != nil {
		t.Fatal(err)
	}
	if m.Amount != 124 || m.Scale != 2 {
		t.Fatalf("rounded scale conversion = %+v", m)
	}
	usd, err := Convert(Money{Amount: 1234, Scale: 2}, Money{Amount: 1, Scale: 0}, Money{Amount: 1, Scale: 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if usd.Amount != 1234 {
		t.Fatalf("USD 1:1 = %+v", usd)
	}
	hkdToUSD, err := Convert(Money{Amount: 10000, Scale: 2}, Money{Amount: 128, Scale: 3}, Money{Amount: 1, Scale: 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if hkdToUSD.Amount != 1280 {
		t.Fatalf("HKD conversion = %+v", hkdToUSD)
	}
	hkdToHKD, err := Convert(
		Money{Amount: 5000, Scale: 2},
		Money{Amount: 12760689, Scale: 8},
		Money{Amount: 12760689, Scale: 8},
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if hkdToHKD.Amount != 5000 {
		t.Fatalf("high precision same-currency conversion = %+v", hkdToHKD)
	}
}
