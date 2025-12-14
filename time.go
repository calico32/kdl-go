package kdl

// This file contains time-related functions from the proposed encoding/json/v2
// package, from github.com/go-json-experiment/json, commit
// d3c622f1b874954c355e60c8e6b6baa5f60d2fed, with some modifications.

// TODO: Use encoding/json/v2 instead (it's available in the standard library as
// of September 2025 via GOEXPERIMENT=jsonv2). Investigate using an on-the-fly struct
// type to use its unmarshaling capabilities for KDL time and duration values
// so we don't have to maintain this code ourselves.

/*!

Copyright (c) 2020 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*/

import (
	"bytes"
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// parseUint is from package encoding/json/v2/internal/jsonwire.

// parseUint parses b as a decimal unsigned integer according to
// a strict subset of the JSON number grammar, returning the value if valid.
// It returns (0, false) if there is a syntax error and
// returns (math.MaxUint64, false) if there is an overflow.
func parseUint(b []byte) (v uint64, ok bool) {
	const unsafeWidth = 20 // len(fmt.Sprint(uint64(math.MaxUint64)))
	var n int
	for ; len(b) > n && ('0' <= b[n] && b[n] <= '9'); n++ {
		v = 10*v + uint64(b[n]-'0')
	}
	switch {
	case n == 0 || len(b) != n || (b[0] == '0' && string(b) != "0"):
		return 0, false
	case n >= unsafeWidth && (b[0] != '1' || v < 1e19 || n > unsafeWidth):
		return math.MaxUint64, false
	}
	return v, true
}

func parseDuration(base uint64, b string) (time.Duration, error) {
	if base == 0 {
		return time.ParseDuration(b)
	} else {
		return parseDurationBase10([]byte(b), base)
	}
}

// parseDurationBase10 parses d from a decimal fractional number,
// where pow10 is a power-of-10 used to scale up the number.
func parseDurationBase10(b []byte, pow10 uint64) (time.Duration, error) {
	suffix, neg := consumeSign(b)                            // consume sign
	wholeBytes, fracBytes := bytesCutByte(suffix, '.', true) // consume whole and frac fields
	whole, okWhole := parseUint(wholeBytes)                  // parse whole field; may overflow
	frac, okFrac := parseFracBase10(fracBytes, pow10)        // parse frac field
	hi, lo := bits.Mul64(whole, uint64(pow10))               // overflow if hi > 0
	sum, co := bits.Add64(lo, uint64(frac), 0)               // overflow if co > 0
	switch d := mayApplyDurationSign(sum, neg); {            // overflow if neg != (d < 0)
	case (!okWhole && whole != math.MaxUint64) || !okFrac:
		return 0, fmt.Errorf("invalid duration %q: %w", b, strconv.ErrSyntax)
	case !okWhole || hi > 0 || co > 0 || neg != (d < 0):
		return 0, fmt.Errorf("invalid duration %q: %w", b, strconv.ErrRange)
	default:
		return d, nil
	}
}

// mayApplyDurationSign inverts n if neg is specified.
func mayApplyDurationSign(n uint64, neg bool) time.Duration {
	if neg {
		return -1 * time.Duration(n)
	} else {
		return +1 * time.Duration(n)
	}
}

func decodeFormat(fmt string) (base uint64, format string) {
	// We assume that an exported constant in the time package will
	// always start with an uppercase ASCII letter.
	base = math.MaxUint // implies custom format
	switch fmt {
	case "ANSIC":
		format = time.ANSIC
	case "UnixDate":
		format = time.UnixDate
	case "RubyDate":
		format = time.RubyDate
	case "RFC822":
		format = time.RFC822
	case "RFC822Z":
		format = time.RFC822Z
	case "RFC850":
		format = time.RFC850
	case "RFC1123":
		format = time.RFC1123
	case "RFC1123Z":
		format = time.RFC1123Z
	case "RFC3339":
		base = 0
		format = time.RFC3339
	case "RFC3339Nano":
		base = 0
		format = time.RFC3339Nano
	case "Kitchen":
		format = time.Kitchen
	case "Stamp":
		format = time.Stamp
	case "StampMilli":
		format = time.StampMilli
	case "StampMicro":
		format = time.StampMicro
	case "StampNano":
		format = time.StampNano
	case "DateTime":
		format = time.DateTime
	case "DateOnly":
		format = time.DateOnly
	case "TimeOnly":
		format = time.TimeOnly
	case "unix":
		base = 1e0
	case "unixmilli":
		base = 1e3
	case "unixmicro":
		base = 1e6
	case "unixnano":
		base = 1e9
	default:
		// Reject any Go identifier in case new constants are supported.
		if strings.TrimFunc(fmt, func(r rune) bool {
			return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
		}) == "" {
			return math.MaxUint, ""
		}
		format = fmt
	}
	return
}

func parseTime(base uint64, format string, b string) (tt time.Time, err error) {
	switch base {
	case 0:
		// Use time.Time.UnmarshalText to avoid possible string allocation.
		err = tt.UnmarshalText([]byte(b))
	case math.MaxUint:
		tt, err = time.Parse(format, b)
	default:
		tt, err = parseTimeUnix([]byte(b), base)
	}
	return
}

// parseTimeUnix parses t formatted as a decimal fractional number,
// where pow10 is a power-of-10 used to scale down the number.
func parseTimeUnix(b []byte, pow10 uint64) (time.Time, error) {
	suffix, neg := consumeSign(b)                            // consume sign
	wholeBytes, fracBytes := bytesCutByte(suffix, '.', true) // consume whole and frac fields
	whole, okWhole := parseUint(wholeBytes)                  // parse whole field; may overflow
	frac, okFrac := parseFracBase10(fracBytes, 1e9/pow10)    // parse frac field
	var sec, nsec int64
	switch {
	case pow10 == 1e0: // fast case where units is in seconds
		sec = int64(whole) // check overflow later after negation
		nsec = int64(frac) // cannot overflow
	case okWhole: // intermediate case where units is not seconds, but no overflow
		sec = int64(whole / pow10)                     // check overflow later after negation
		nsec = int64((whole%pow10)*(1e9/pow10) + frac) // cannot overflow
	case !okWhole && whole == math.MaxUint64: // slow case where units is not seconds and overflow occurred
		width := int(math.Log10(float64(pow10)))                               // compute len(strconv.Itoa(pow10-1))
		whole, okWhole = parseUint(wholeBytes[:len(wholeBytes)-width])         // parse the upper whole field
		mid, _ := parsePaddedBase10(wholeBytes[len(wholeBytes)-width:], pow10) // parse the lower whole field
		sec = int64(whole)                                                     // check overflow later after negation
		nsec = int64(mid*(1e9/pow10) + frac)                                   // cannot overflow
	}
	if neg {
		sec, nsec = negateSecNano(sec, nsec)
	}
	switch t := time.Unix(sec, nsec).UTC(); {
	case (!okWhole && whole != math.MaxUint64) || !okFrac:
		return time.Time{}, fmt.Errorf("invalid time %q: %w", b, strconv.ErrSyntax)
	case !okWhole || neg != (t.Unix() < 0):
		return time.Time{}, fmt.Errorf("invalid time %q: %w", b, strconv.ErrRange)
	default:
		return t, nil
	}
}

// negateSecNano negates a Unix timestamp, where nsec must be within [0, 1e9).
func negateSecNano(sec, nsec int64) (int64, int64) {
	sec = ^sec               // twos-complement negation (i.e., -1*sec + 1)
	nsec = -nsec + 1e9       // negate nsec and add 1e9 (which is the extra +1 from sec negation)
	sec += int64(nsec / 1e9) // handle possible overflow of nsec if it started as zero
	nsec %= 1e9              // ensure nsec stays within [0, 1e9)
	return sec, nsec
}

// parseFracBase10 parses the fraction of n/max10,
// where max10 is a power-of-10 that is larger than n.
func parseFracBase10(b []byte, max10 uint64) (n uint64, ok bool) {
	switch {
	case len(b) == 0:
		return 0, true
	case len(b) < len(".0") || b[0] != '.':
		return 0, false
	}
	return parsePaddedBase10(b[len("."):], max10)
}

// parsePaddedBase10 parses b as the zero-padded encoding of n,
// where max10 is a power-of-10 that is larger than n.
// Truncated suffix is treated as implicit zeros.
// Extended suffix is ignored, but verified to contain only digits.
func parsePaddedBase10(b []byte, max10 uint64) (n uint64, ok bool) {
	pow10 := uint64(1)
	for pow10 < max10 {
		n *= 10
		if len(b) > 0 {
			if b[0] < '0' || '9' < b[0] {
				return n, false
			}
			n += uint64(b[0] - '0')
			b = b[1:]
		}
		pow10 *= 10
	}
	if len(b) > 0 && len(bytes.TrimRight(b, "0123456789")) > 0 {
		return n, false // trailing characters are not digits
	}
	return n, true
}

// consumeSign consumes an optional leading negative sign.
func consumeSign(b []byte) ([]byte, bool) {
	if len(b) > 0 && b[0] == '-' {
		return b[len("-"):], true
	}
	return b, false
}

// bytesCutByte is similar to bytes.Cut(b, []byte{c}),
// except c may optionally be included as part of the suffix.
func bytesCutByte(b []byte, c byte, include bool) ([]byte, []byte) {
	if i := bytes.IndexByte(b, c); i >= 0 {
		if include {
			return b[:i], b[i:]
		}
		return b[:i], b[i+1:]
	}
	return b, nil
}
