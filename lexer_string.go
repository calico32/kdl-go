package kdl

import (
	"strings"
)

func (l *lexer) readQuotedString() token {
	start := l.offset
	l.next() // consume opening quote
	if l.ch == '"' && l.peek() == '"' {
		// three quotes, multi-line string
		l.next() // second quote
		l.next() // third quote

		// 3.12 Multi-Line String
		// https://kdl.dev/spec/#name-multi-line-string
		// Its first line MUST immediately start with a Newline (Section 3.18)
		// after its opening """.
		if !isNewline(l.ch) {
			l.errorf(start, "multi-line string opening must be followed by a newline")
		} else {
			l.readNewline()
		}

		lines := []string{}
		lineStart := l.offset
		for {
			for l.ch != runeEOF && !isNewline(l.ch) && !l.match(`"""`) {
				l.readStringChar(true)
			}
			if l.ch == runeEOF {
				l.errorf(start, "unterminated multi-line string")
				return l.tok(tokenIllegal, start, l.text(start, l.offset))
			}
			// 3.12. Whitespace escapes are resolved before dedentation, and
			// all other escapes after it.
			line := removeWhitespaceEscapes(l.text(lineStart, l.offset))
			if isNewline(l.ch) {
				lines = append(lines, line)
				l.readNewline()
				lineStart = l.offset
				continue
			}

			// consume closing """
			l.next() // first "
			l.next() // second "
			l.next() // third "

			prefix := line
			prefixHasNonwhitespace := false
			for _, ch := range prefix {
				if !isUnicodeSpace(ch) {
					prefixHasNonwhitespace = true
					break
				}
			}
			if prefixHasNonwhitespace {
				l.errorf(lineStart, "only whitespace allowed on final line of multi-line string")
			}
			var content strings.Builder
			content.Grow(int(l.offset - start))
			for i, ln := range lines {
				hasNonwhitespaceChars := false
				for _, ch := range ln {
					if !isUnicodeSpace(ch) {
						hasNonwhitespaceChars = true
						break
					}
				}
				if hasNonwhitespaceChars && !strings.HasPrefix(ln, prefix) {
					l.errorf(lineStart, "line %d missing required leading whitespace", i+1)
				}
				if i > 0 {
					content.WriteString("\n")
				}
				if hasNonwhitespaceChars {
					content.WriteString(strings.TrimPrefix(ln, prefix))
				}
			}
			unescaped, err := unescapeString(content.String(), l.version)
			if err != nil {
				l.errorf(start, "invalid multi-line string: %s", err)
				return l.tok(tokenQuotedMultiLineString, start, "")
			}
			return l.tok(tokenQuotedMultiLineString, start, unescaped)
		}
	}

	// single-line quoted string
	for !l.readStringChar(false) {
	}
	content, err := unescapeString(l.text(start+1, l.offset-1), l.version)
	if err != nil {
		l.errorf(start, "invalid string: %s", err)
	}
	return l.tok(tokenQuotedString, start, content)
}

func (l *lexer) readStringChar(multiline bool) (terminal bool) {
	switch ch := l.ch; {
	case ch == '\\':
		l.next()
		switch l.ch {
		case '"', '\\', 'b', 'f', 'n', 'r', 't', 's':
			l.next()
			return false
		case '/':
			if l.version == Version1 {
				// OK
				l.next()
				return false
			}
			l.errorf(l.offset, "invalid escape sequence in string: \\/ (v1 only)")
			return false
		case 'u':
			// \u{hex-unicode}
			// hex-unicode := hex-digit{1, 6} - surrogates
			// surrogates := [dD][8-9a-fA-F]hex-digit{2}
			l.next() // consume 'u'
			if l.ch != '{' {
				l.errorf(l.offset, "expected '{' after \\u in string escape")
				return false
			}
			l.next() // consume '{'
			hexStart := l.offset
			hexDigits := 0
			for isHexDigit(l.ch) {
				hexDigits++
				l.next() // consume hex digit
			}
			if hexDigits == 0 || hexDigits > 6 {
				l.errorf(hexStart, "expected 1-6 hex digits in \\u{} escape")
				return false
			}
			if l.ch != '}' {
				l.errorf(l.offset, "expected '}' to close \\u{} escape")
				return false
			}
			l.next()

			// check for surrogate code points
			hex := l.text(hexStart, l.offset)
			if len(hex) == 4 && (hex[0] == 'd' || hex[0] == 'D') && (hex[1] == '8' || hex[1] == '9' || ('a' <= hex[1] && hex[1] <= 'f') || ('A' <= hex[1] && hex[1] <= 'F')) {
				l.errorf(hexStart, "invalid surrogate code point in \\u{} escape")
			}

			return false
		}

		// whitespace escape
		if isUnicodeSpace(l.ch) || isNewline(l.ch) {
			for isUnicodeSpace(l.ch) || isNewline(l.ch) {
				l.next()
			}
			return false
		}

		// invalid escape
		l.errorf(l.offset, "invalid escape sequence in string: \\%c", l.ch)
		l.next() // skip invalid escape char
		return false

	case ch == '"':
		// closing quote in single-line string
		l.next()
		return !multiline

	case ch == runeEOF:
		l.errorf(l.offset-1, "unterminated string")
		return true

	case isNewline(ch):
		l.next()
		if multiline || l.version == Version1 {
			return false
		}

		l.errorf(l.offset, "unexpected newline in single-line string")
		return true

	default:
		// regular character
		l.next()
		return false
	}
}

// removeWhitespaceEscapes removes the whitespace escapes from s, along with
// the whitespace and newlines they escape, and leaves all other escapes as
// they are.
func removeWhitespaceEscapes(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var result strings.Builder
	result.Grow(len(s))
	escaped := false
	inWhitespace := false
	for _, ch := range s {
		if inWhitespace {
			if isUnicodeSpace(ch) || isNewline(ch) {
				continue
			}
			inWhitespace = false
		}
		if escaped {
			escaped = false
			if isUnicodeSpace(ch) || isNewline(ch) {
				inWhitespace = true
				continue
			}
			result.WriteRune('\\')
			result.WriteRune(ch)
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		result.WriteRune(ch)
	}
	if escaped {
		result.WriteRune('\\')
	}
	return result.String()
}
