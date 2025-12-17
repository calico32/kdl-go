package kdl

import (
	"strings"
)

func (l *lexer) readRawString() token {
	// first '#' has already been consumed
	start := l.offset - 1
	hashCount := 1
	for l.ch == '#' {
		hashCount++
		l.next()
	}

	if l.ch != '"' {
		l.errorf(start, "invalid raw string (expected '\"' after %d '#', got %q)", hashCount, l.ch)
		return token{tokenIllegal, start, l.text(start, l.offset)}
	}
	l.next() // first quote

	if l.ch == '"' && l.peek() == '"' {
		// three quotes, raw multi-line string
		l.next() // second quote
		l.next() // third quote

		closingSeq := `"""` + strings.Repeat("#", hashCount)

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
			for l.ch != runeEOF && !isNewline(l.ch) && !l.match(closingSeq) {
				l.next()
			}
			if l.ch == runeEOF {
				l.errorf(start, "unterminated multi-line raw string")
				return token{tokenIllegal, start, l.text(start, l.offset)}
			}
			line := l.text(lineStart, l.offset)
			if isNewline(l.ch) {
				lines = append(lines, line)
				l.next()
				lineStart = l.offset
				continue
			}

			// just before closing sequence

			// 3.12. Its final line MUST contain only whitespace before the
			// closing """.
			hasNonwhitespaceChars := false
			for _, ch := range line {
				if !isUnicodeSpace(ch) {
					hasNonwhitespaceChars = true
					break
				}
			}
			if hasNonwhitespaceChars {
				l.errorf(lineStart, "only whitespace allowed before closing sequence of multi-line raw string")
			}
			// consume closing sequence
			for range closingSeq {
				l.next()
			}
			// 3.12. All in-between lines that contain non-newline,
			// non-whitespace characters MUST start with at least the exact same
			// whitespace as the final line (precisely matching codepoints, not
			// merely counting characters or "size"); they may contain
			// additional whitespace following this prefix.
			prefix := line
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
					l.errorf(lineStart, "line missing required leading whitespace")
				}
				if i > 0 {
					content.WriteString("\n")
				}
				if hasNonwhitespaceChars {
					content.WriteString(strings.TrimPrefix(ln, prefix))
				}
			}
			return token{tokenRawMultiLineString, start, content.String()}
		}

	}

	// single-line raw string
	// contentStart := l.offset
	closingSeq := `"` + strings.Repeat("#", hashCount)
	for l.ch != runeEOF && !isNewline(l.ch) && !l.match(closingSeq) {
		l.next()
	}
	if l.ch == runeEOF {
		l.errorf(start, "unterminated raw string")
		return token{tokenIllegal, start, l.text(start, l.offset)}
	}
	if isNewline(l.ch) {
		l.errorf(start, "unexpected newline in single-line raw string")
		return token{tokenIllegal, start, l.text(start, l.offset)}
	}

	// TODO: necessary? if there are three quotes in a row, it would have been
	// detected as a multi-line string earlier
	// content := l.text(contentStart, l.offset)
	// if content == `"` {
	// 	l.errorf(start, "single-line raw strings cannot look like multi-line ones")
	// }

	// consume closing sequence
	for i := 0; i < len(closingSeq); i++ {
		l.next()
	}
	return token{tokenRawString, start, l.text(start+Pos(len(closingSeq)), l.offset-Pos(len(closingSeq)))}
}
