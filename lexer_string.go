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
				return token{tokenIllegal, start, l.text(start, l.offset)}
			}
			line := l.text(lineStart, l.offset)
			if isNewline(l.ch) {
				lines = append(lines, line)
				l.next()
				lineStart = l.offset
				continue
			}

			lines = append(lines, line)
			// consume closing """
			l.next() // first "
			l.next() // second "
			l.next() // third "

			var content strings.Builder
			content.Grow(int(l.offset - start))
			for i, ln := range lines {
				if i > 0 {
					content.WriteString("\n")
				}
				content.WriteString(ln)
			}
			unescaped, err := unescapeString(l.version, content.String())
			if err != nil {
				l.errorf(start, "invalid multi-line string: %s", err)
			}
			content.Reset()
			unescapedLines := strings.Split(unescaped, "\n")
			if len(lines) == 0 {
				return token{tokenQuotedMultiLineString, start, unescaped}
			}

			prefix := unescapedLines[len(unescapedLines)-1]
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
			for i, ln := range lines[:len(lines)-1] {
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
					content.WriteString(strings.TrimPrefix(unescapedLines[i], prefix))
				}
			}
			return token{tokenQuotedMultiLineString, start, content.String()}
		}
	}

	// single-line quoted string
	for !l.readStringChar(false) {
	}
	content, err := unescapeString(l.version, l.text(start+1, l.offset-1))
	if err != nil {
		l.errorf(start, "invalid string: %s", err)
	}
	return token{tokenQuotedString, start, content}
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
