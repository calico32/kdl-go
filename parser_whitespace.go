package kdl

import "strings"

// skipWS skips whitespace and multi-line comments until a different token is
// found.
//
//	ws := unicode-space | multi-line-comment
func (p *parser) skipWS() {
	for {
		switch p.token.Type {
		case tokenWS:
			p.next()
		case tokenMultiLineCommentStart:
			p.readMultiLineComment()
		default:
			return
		}
	}
}

// skipLineSpace reads line spaces until a different token is found.
//
//	line-space := node-space | newline | single-line-comment
func (p *parser) skipLineSpace() {
	for {
		switch p.token.Type {
		case tokenNewline, tokenSingleLineComment, tokenWS, tokenBackslash, tokenMultiLineCommentStart:
			p.readLineSpace()
		default:
			return
		}
	}
}

// skipLineSpaceBlank is like skipLineSpace but returns true if a blank line was
// encountered (two or more consecutive newlines, ignoring comments and whitespace).
func (p *parser) skipLineSpaceBlank() bool {
	return p.skipLineSpaceBlankFrom(0)
}

// skipLineSpaceBlankFrom is like skipLineSpaceBlank but starts the newline
// count at initialNewlines. Pass 1 when the caller already consumed a newline
// terminator so that a single additional newline correctly signals a blank line.
func (p *parser) skipLineSpaceBlankFrom(initialNewlines int) bool {
	newlines := initialNewlines
	for {
		switch p.token.Type {
		case tokenNewline:
			newlines++
			p.next()
		case tokenSingleLineComment, tokenWS, tokenBackslash, tokenMultiLineCommentStart:
			p.readLineSpace()
		default:
			return newlines >= 2
		}
	}
}

// readLineSpace reads one line space.
//
//	line-space := node-space | newline | single-line-comment
func (p *parser) readLineSpace() {
	switch p.token.Type {
	case tokenNewline, tokenSingleLineComment:
		p.next()
	case tokenWS, tokenBackslash, tokenMultiLineCommentStart:
		p.readNodeSpace()
	default:
		p.errorExpected("line space")
	}
}

// skipNodeSpace reads node spaces until a different token is found.
//
//	node-space := ws* escline ws* | ws+
func (p *parser) skipNodeSpace() {
	for {
		switch p.token.Type {
		case tokenWS, tokenBackslash, tokenMultiLineCommentStart:
			p.readNodeSpace()
		default:
			return
		}
	}
}

// readNodeSpace reads one node space.
//
//	node-space := ws* escline ws* | ws+
func (p *parser) readNodeSpace() bool {
	switch p.token.Type {
	case tokenWS, tokenMultiLineCommentStart:
		p.skipWS()
		if p.token.Type != tokenBackslash {
			return true
		}
		fallthrough
	case tokenBackslash:
		p.readEscline()
		p.skipWS()
		return true
	default:
		p.errorExpected("node space")
		return false
	}
}

// readEscline reads an escline.
//
//	escline := '\\' ws* (single-line-comment | newline | eof)
func (p *parser) readEscline() {
	p.next() // consume Backslash
	p.skipWS()
	switch p.token.Type {
	case tokenNewline, tokenSingleLineComment:
		p.next()
	case tokenEOF:
		// OK, but don't consume
	default:
		p.errorExpected("end of line after escline")
	}
}

// readMultiLineComment reads a multi-line comment.
//
//	multi-line-comment := '/*' commented-block
//	commented-block := '*/' | (multi-line-comment | '*' | '/' | [^*/]+) commented-block
func (p *parser) readMultiLineComment() {
	p.next() // consume MultiLineCommentStart
	x := 1
	for x > 0 && p.token.Type != tokenEOF {
		switch p.token.Type {
		case tokenMultiLineCommentStart:
			x++
		case tokenMultiLineCommentEnd:
			x--
		}
		p.next() // consume start/content/end
	}
	if x > 0 {
		p.errorf(p.token.Pos, "unterminated multi-line comment")
	}
}

// readMultiLineCommentText reads a multi-line comment and returns its full raw text
// (e.g. "/* foo /* nested */ bar */"), including the opening "/*".
func (p *parser) readMultiLineCommentText() string {
	var sb strings.Builder
	sb.WriteString("/*")
	p.next() // consume tokenMultiLineCommentStart
	depth := 1
	for depth > 0 && p.token.Type != tokenEOF {
		sb.WriteString(p.token.Text)
		switch p.token.Type {
		case tokenMultiLineCommentStart:
			depth++
		case tokenMultiLineCommentEnd:
			depth--
		}
		p.next()
	}
	if depth > 0 {
		p.errorf(p.token.Pos, "unterminated multi-line comment")
	}
	return sb.String()
}

// collectBetweenNodes collects single-line and multi-line comments from line-space,
// skipping whitespace and tracking newlines. initialNewlines is the count of newlines
// already consumed before this call (pass 1 when the caller already consumed a newline
// terminator, so that one more newline correctly signals a blank line).
// Returns the collected comments and whether a blank line (≥2 consecutive newlines) was seen.
func (p *parser) collectBetweenNodes(initialNewlines int) (comments []Comment, blankLine bool) {
	newlines := initialNewlines
	for {
		switch p.token.Type {
		case tokenNewline:
			newlines++
			p.next()
		case tokenWS:
			p.next()
		case tokenBackslash:
			p.readEscline()
		case tokenSingleLineComment:
			c := Comment{kind: CommentSingleLine, text: p.token.Text}
			if p.withLocations {
				c.start = p.lexer.File().Location(p.token.Pos)
				c.end = p.lexer.File().Location(p.token.EndPos)
			}
			comments = append(comments, c)
			p.next()
		case tokenMultiLineCommentStart:
			startPos := p.token.Pos
			text := p.readMultiLineCommentText()
			c := Comment{kind: CommentMultiLine, text: text}
			if p.withLocations {
				c.start = p.lexer.File().Location(startPos)
				c.end = p.lexer.File().Location(startPos + Pos(len(text)))
			}
			comments = append(comments, c)
			// Consume the newline that ends the comment's own line without
			// counting it toward blank-line detection. A subsequent newline
			// (after this consumed one) is the real blank-line signal.
			if p.token.Type == tokenNewline {
				p.next()
			}
		default:
			return comments, newlines >= 2
		}
	}
}
