package kdl

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

func isUnicodeSpace(ch rune) bool {
	// 3.17. Whitespace
	// https://kdl.dev/spec/#section-3.17
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
	switch ch {
	// CRLF case is handled elsewhere
	case 0x000D, // Carriage Return
		0x000A, // Line Feed
		0x0085, // Next Line
		0x000B, // Vertical Tab
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
