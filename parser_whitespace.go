package kdl

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
