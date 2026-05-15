package kdl

// CanBeBareIdentifier reports whether s can be written as an unquoted
// identifier (or value-as-identifier) under the given KDL spec version.
// The empty string and reserved keywords (true/false/null/inf/-inf/nan in v2)
// always require quoting. version must be Version1 or Version2; VersionAuto
// is treated as Version2.
func CanBeBareIdentifier(s string, version Version) bool {
	if s == "" {
		return false
	}
	if version == Version1 {
		runes := []rune(s)
		allowDash := len(runes) == 1 || !isDigit(runes[1])
		for i, r := range runes {
			if i == 0 {
				if !isV1IdentStartChar(r, allowDash) {
					return false
				}
			} else if !isV1IdentChar(r) {
				return false
			}
		}
		return true
	}
	switch s {
	case "true", "false", "null", "inf", "-inf", "nan":
		return false
	}
	for i, r := range s {
		if i == 0 {
			// The following characters cannot be the first character in an
			// Identifier String (Section 3.10):
			//  - Any decimal digit (0-9)
			//  - Any non-identifier characters (Section 3.10.2)
			// Additionally, the following initial characters impose limitations
			// on subsequent characters:
			//  - the + and - characters can only be used as an initial character
			//    if the second character is not a digit. If the second character
			//    is ., then the third character must not be a digit.
			//  - the . character can only be used as an initial character if the
			//    second character is not a digit.
			if isDigit(r) || !isIdentChar(r) {
				return false
			}
			if isSign(r) && len(s) > 1 {
				if isDigit(rune(s[1])) {
					return false
				}
				if s[1] == '.' && len(s) > 2 && isDigit(rune(s[2])) {
					return false
				}
			}
			if r == '.' && len(s) > 1 && isDigit(rune(s[1])) {
				return false
			}
		} else if !isIdentChar(r) {
			return false
		}
	}
	return true
}

// v2 only
func isIdentStartChar(ch rune) bool {
	return isIdentChar(ch) && !isDigit(ch) && !isSign(ch) && ch != '.'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isLetter(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isSign(ch rune) bool {
	return ch == '+' || ch == '-'
}

// v2 only
func isIdentChar(ch rune) bool {
	if isUnicodeSpace(ch) || isNewline(ch) || isDisallowedChar(ch) {
		return false
	}

	switch ch {
	case '\\', '/', '(', ')', '{', '}', ';', '[', ']', '"', '#', '=':
		return false
	default:
		return true
	}
}

func isV1IdentStartChar(ch rune, allowSign bool) bool {
	// https://kdl.dev/spec-v1/#non-initial-characters
	if '0' <= ch && ch <= '9' {
		return false
	}
	if !allowSign && isSign(ch) {
		return false
	}
	return isV1IdentChar(ch)
}

func isV1IdentChar(ch rune) bool {
	// https://kdl.dev/spec-v1/#non-identifier-characters
	if ch <= 0x20 || ch >= 0x10FFFF {
		return false
	}
	if isUnicodeSpace(ch) || isNewline(ch) {
		return false
	}
	switch ch {
	case '\\', '/', '(', ')', '{', '}', '<', '>', ';', '[', ']', '=', ',', '"':
		return false
	default:
		return true
	}
}

func isUnicodeSpace(ch rune) bool {
	// 3.17. Whitespace
	// https://kdl.dev/spec/#section-3.17
	// https://kdl.dev/spec-v1/#whitespace (same as v2)
	switch ch {
	case 0x0009, // Character Tabulation
		0x0020, // Space
		0x00A0, // No-Break Space
		0x1680, // Ogham Space Mark
		0x2000, // En Quad
		0x2001, // Em Quad
		0x2002, // En Space
		0x2003, // Em Space
		0x2004, // Three-Per-Em Space
		0x2005, // Four-Per-Em Space
		0x2006, // Six-Per-Em Space
		0x2007, // Figure Space
		0x2008, // Punctuation Space
		0x2009, // Thin Space
		0x200A, // Hair Space
		0x202F, // Narrow No-Break Space
		0x205F, // Medium Mathematical Space
		0x3000: // Ideographic Space
		return true
	default:
		return false
	}
}

func isNewline(ch rune) bool {
	// 3.18. Newline
	// https://kdl.dev/spec/#section-3.18
	// https://kdl.dev/spec-v1/#newline (same as v2 except for vertical tab U+000B)
	switch ch {
	// CRLF case is handled elsewhere
	case 0x000D, // Carriage Return
		0x000A, // Line Feed
		0x0085, // Next Line
		0x000B, // Vertical Tab (v2 only but was supposed to be in v1, probably okay to include anyway)
		0x000C, // Form Feed
		0x2028, // Line Separator
		0x2029: // Paragraph Separator
		return true
	default:
		return false
	}
}

func isDisallowedChar(ch rune) bool {
	// 3.19. Disallowed Literal Code Points
	// https://kdl.dev/spec/#section-3.19
	return 0x0000 <= ch && ch <= 0x0008 || // various control characters
		0x000E <= ch && ch <= 0x001F || // various control characters
		ch == 0x007F || // delete control character
		0xD800 <= ch && ch <= 0xDFFF || // surrogate halves (not a scalar value)
		0x200E <= ch && ch <= 0x200F || // bidi control characters
		0x202A <= ch && ch <= 0x202E || // bidi control characters
		0x2066 <= ch && ch <= 0x2069 || // bidi control characters
		ch == 0xFEFF // zero width no-break space/BOM
}
