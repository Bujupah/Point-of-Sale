// Package domain holds types shared across every module: money, common errors,
// and the snapshot-friendly value objects that sale records are built from.
package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is an integer amount in minor currency units (cents for EUR/USD,
// thousandths for 3-decimal currencies). Never use float64 for currency.
type Money int64

func (m Money) Add(other Money) Money      { return m + other }
func (m Money) Sub(other Money) Money      { return m - other }
func (m Money) Neg() Money                 { return -m }
func (m Money) IsZero() bool               { return m == 0 }
func (m Money) IsNegative() bool           { return m < 0 }
func (m Money) Mul(qty int64) Money        { return Money(int64(m) * qty) }

// MulPercentBps multiplies by a percentage expressed in basis points
// (10000 = 100.00%), rounding half-up to the nearest minor unit.
func (m Money) MulPercentBps(bps int64) Money {
	num := int64(m) * bps
	den := int64(10000)
	if num < 0 {
		return Money(-((-num + den/2) / den))
	}
	return Money((num + den/2) / den)
}

// Format renders the amount using the given number of decimal places
// (2 for most currencies, 3 for currencies like KWD/BHD/OMR).
func (m Money) Format(decimals int) string {
	neg := m < 0
	v := int64(m)
	if neg {
		v = -v
	}
	div := int64(1)
	for i := 0; i < decimals; i++ {
		div *= 10
	}
	whole := v / div
	frac := v % div
	s := strconv.FormatInt(whole, 10)
	if decimals > 0 {
		fracStr := strconv.FormatInt(frac, 10)
		fracStr = strings.Repeat("0", decimals-len(fracStr)) + fracStr
		s = s + "." + fracStr
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ParseMoney parses a decimal string (e.g. "12.50") into minor units given
// the currency's decimal precision.
func ParseMoney(s string, decimals int) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > decimals {
		frac = frac[:decimals]
	}
	for len(frac) < decimals {
		frac += "0"
	}
	if whole == "" {
		whole = "0"
	}
	combined := whole + frac
	v, err := strconv.ParseInt(combined, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	if neg {
		v = -v
	}
	return Money(v), nil
}
