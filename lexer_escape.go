package kdl

import (
	"fmt"
	"strconv"
	"strings"
)

func unescapeString(s string, v Version) (string, error) {
	type unescapeState int
	const (
		normal unescapeState = iota
		escape
		unicode
		unicodeHex
		whitespace
	)
	var result strings.Builder
	state := normal
	unicodeStart := -1

	for i, ch := range s {
		if state == unicode {
			if ch != '{' {
				return "", fmt.Errorf("invalid unicode escape sequence: missing '{'")
			}
			state = unicodeHex
			unicodeStart = i + 1
			continue
		}
		if state == unicodeHex {
			if ch == '}' {
				u := s[unicodeStart:i]
				r, err := strconv.ParseInt(u, 16, 32)
				if err != nil {
					return "", fmt.Errorf("invalid unicode escape sequence: %w", err)
				}
				if r >= 0xD800 && r <= 0xDFFF {
					return "", fmt.Errorf("invalid unicode escape sequence: surrogate code point U+%04X", r)
				}
				if r > 0x10FFFF {
					return "", fmt.Errorf("invalid unicode escape sequence: code point U+%X out of range", r)
				}
				result.WriteRune(rune(r))
				state = normal
				continue
			}
			if !isHexDigit(ch) {
				return "", fmt.Errorf("invalid unicode escape sequence: invalid character '%c' in unicode escape", ch)
			}
			continue
		}
		if state == whitespace {
			if isUnicodeSpace(ch) || isNewline(ch) {
				continue
			}
			// end of whitespace
			state = normal
			if ch == '\\' {
				state = escape
			} else {
				result.WriteRune(ch)
			}
			continue
		}

		if state == escape {
			switch ch {
			case 'n':
				result.WriteRune(0x000A)
			case 'r':
				result.WriteRune(0x000D)
			case 't':
				result.WriteRune(0x0009)
			case '\\':
				result.WriteRune(0x005C)
			case '"':
				result.WriteRune(0x0022)
			case 'b':
				result.WriteRune(0x0008)
			case 'f':
				result.WriteRune(0x000C)
			case 's':
				result.WriteRune(0x0020)
			case 'u':
				state = unicode
				continue
			default:
				if ch == '/' && v == Version1 {
					result.WriteRune(0x002F)
					break
				}
				if isUnicodeSpace(ch) || isNewline(ch) {
					state = whitespace
					continue
				}
				return "", fmt.Errorf("invalid escape sequence: \\%c", ch)
			}

			state = normal
			continue
		}

		if ch == '\\' {
			state = escape
			continue
		}

		result.WriteRune(ch)
	}

	if state == escape {
		return "", fmt.Errorf("invalid escape sequence: trailing backslash")
	}
	if state == unicodeHex {
		return "", fmt.Errorf("invalid unicode escape sequence: missing '}'")
	}
	return result.String(), nil
}

// EscapeString returns the escaped form of s, suitable for use in a KDL string
// literal. The result is not wrapped in quotes.
func EscapeString(s string, v Version) string {
	var result strings.Builder
	for _, ch := range s {
		switch ch {
		case 0x000A:
			result.WriteString(`\n`)
		case 0x000D:
			result.WriteString(`\r`)
		case 0x0009:
			result.WriteString(`\t`)
		case 0x005C:
			result.WriteString(`\\`)
		case 0x0022:
			result.WriteString(`\"`)
		case 0x0008:
			result.WriteString(`\b`)
		case 0x000C:
			result.WriteString(`\f`)
		case 0x0020:
			result.WriteString(` `) // no need to escape space
		default:
			if isDisallowedChar(ch) {
				fmt.Fprintf(&result, `\u{%X}`, ch)
			} else {
				result.WriteRune(ch)
			}
		}
	}
	return result.String()
}
