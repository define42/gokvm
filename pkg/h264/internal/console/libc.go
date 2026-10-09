package console

import (
	"math"
	"strconv"
	"strings"
)

func isCSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

// Strtol is strtol (s, NULL, 10) as used by atoi/atol: leading white space
// is skipped, an optional sign and the longest run of decimal digits are
// parsed, and the result saturates at the int64 (LP64 long) limits.
func Strtol(s string) int64 {
	i := 0
	for i < len(s) && isCSpace(s[i]) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var v uint64
	overflow := false
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		d := uint64(s[i] - '0')
		if v > (math.MaxUint64-d)/10 {
			overflow = true
			continue
		}
		v = v*10 + d
	}
	if neg {
		if overflow || v > 1<<63 {
			return math.MinInt64
		}
		return -int64(v)
	}
	if overflow || v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// Atoi is the C atoi(): (int) strtol (s, NULL, 10).
func Atoi(s string) int32 {
	return int32(Strtol(s))
}

// Atol is the C atol() on an LP64 platform.
func Atol(s string) int64 {
	return Strtol(s)
}

// Atof is the C atof(): strtod (s, NULL). It parses the longest prefix of
// s (after leading white space) that forms a decimal floating point number,
// an "inf"/"infinity" or a "nan"; 0 is returned when there is none.
func Atof(s string) float64 {
	i := 0
	for i < len(s) && isCSpace(s[i]) {
		i++
	}
	s = s[i:]
	j := 0
	if j < len(s) && (s[j] == '+' || s[j] == '-') {
		j++
	}
	rest := strings.ToLower(s[j:])
	switch {
	case strings.HasPrefix(rest, "infinity"):
		v, _ := strconv.ParseFloat(s[:j+8], 64)
		return v
	case strings.HasPrefix(rest, "inf"):
		v, _ := strconv.ParseFloat(s[:j+3], 64)
		return v
	case strings.HasPrefix(rest, "nan"):
		return math.NaN()
	}
	if strings.HasPrefix(rest, "0x") {
		// hexadecimal floating point: find the longest parsable prefix.
		for k := len(s); k > j+2; k-- {
			if v, err := strconv.ParseFloat(fixHexFloat(s[:k]), 64); err == nil || isRangeErr(err) {
				return v
			}
		}
		return 0
	}
	digits := 0
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
		digits++
	}
	if j < len(s) && s[j] == '.' {
		j++
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
			digits++
		}
	}
	if digits == 0 {
		return 0
	}
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		k := j + 1
		if k < len(s) && (s[k] == '+' || s[k] == '-') {
			k++
		}
		if k < len(s) && s[k] >= '0' && s[k] <= '9' {
			for k < len(s) && s[k] >= '0' && s[k] <= '9' {
				k++
			}
			j = k
		}
	}
	v, _ := strconv.ParseFloat(s[:j], 64) // a range error yields ±Inf or 0 like strtod
	return v
}

// fixHexFloat appends a zero binary exponent when it is missing, as Go
// requires the 'p' exponent for hexadecimal floats.
func fixHexFloat(s string) string {
	if strings.ContainsAny(s, "pP") {
		return s
	}
	return s + "p0"
}

func isRangeErr(err error) bool {
	ne, ok := err.(*strconv.NumError)
	return ok && ne.Err == strconv.ErrRange
}
