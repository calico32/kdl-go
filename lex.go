package kdl

func (l *lexer) lexDefault() token {
	start := l.offset
	switch ch := l.ch; {
	case ch == runeEOF:
		return token{tokenEOF, l.offset, ""}
	case isIdentStartChar(ch):
		l.next()
		for isIdentChar(l.ch) {
			l.next()
		}
		lit := l.text(start, l.offset)
		switch lit {
		case "true", "false", "null", "inf", "-inf", "nan":
			l.errorf(start, "invalid identifier: %q", lit)
		}
		return token{tokenUnambiguousIdent, start, lit}
	case isUnicodeSpace(ch):
		l.next()
		for isUnicodeSpace(l.ch) {
			l.next()
		}
		return token{tokenWS, start, l.text(start, l.offset)}
	case isNewline(ch):
		return l.readNewline()
	case ch == '/':
		l.next()
		if l.ch == '*' {
			l.next()
			l.pushMode(modeMultiLineComment)
			return token{tokenMultiLineCommentStart, start, "/*"}
		}
		if l.ch == '/' {
			l.next()
			for l.ch != runeEOF && !isNewline(l.ch) {
				l.next()
			}
			if isNewline(l.ch) {
				l.next()
			}
			return token{tokenSingleLineComment, start, l.text(start, l.offset)}
		}
		if l.ch == '-' {
			l.next()
			return token{tokenSlashdash, start, "/-"}
		}

	case ch == '\\':
		l.next()
		return token{tokenBackslash, start, "\\"}

	case ch == '{':
		l.next()
		return token{tokenLBrace, start, "{"}
	case ch == '}':
		l.next()
		return token{tokenRBrace, start, "}"}
	case ch == '(':
		l.next()
		return token{tokenLParen, start, "("}
	case ch == ')':
		l.next()
		return token{tokenRParen, start, ")"}
	case ch == ';':
		l.next()
		return token{tokenSemi, start, ";"}
	case ch == '=':
		l.next()
		return token{tokenEqual, start, "="}

	case ch == '#':
		l.next()
		if l.ch == '-' || isLetter(l.ch) {
			l.next()
			for isLetter(l.ch) {
				l.next()
			}
			lit := l.text(start, l.offset)
			switch lit {
			case "#inf":
				return token{tokenInf, start, lit}
			case "#-inf":
				return token{tokenNegInf, start, lit}
			case "#nan":
				return token{tokenNaN, start, lit}
			case "#true":
				return token{tokenTrue, start, lit}
			case "#false":
				return token{tokenFalse, start, lit}
			case "#null":
				return token{tokenNull, start, lit}
			default:
				l.errorf(start, "invalid keyword: %q", lit)
				return token{tokenIllegal, start, lit}
			}
		}

		return l.readRawString()

	case ch == '"':
		return l.readQuotedString()

	case isSign(ch) && isDigit(rune(l.peek())):
		return l.readNumber()

	case isDigit(ch):
		return l.readNumber()

	case isSign(ch) && l.peek() == '.':
		// dotted-ident := sign? '.' ((identifier-char - digit) identifier-char*)?
		l.next() // consume sign
		fallthrough
	case ch == '.':
		l.next() // consume '.'
		if isDigit(l.ch) {
			// invalid number - integer part is missing
			l.errorf(start, "invalid number: missing integer part before decimal point")
			return token{tokenIllegal, start, l.text(start, l.offset)}
		}
		for isIdentChar(l.ch) {
			l.next()
		}
		return token{tokenDottedIdent, start, l.text(start, l.offset)}

	case isSign(ch):
		// signed-ident := sign ((identifier-char - digit - '.') identifier-char*)?
		l.next() // consume sign
		if isDigit(l.ch) {
			// invalid number - integer part is missing
			l.errorf(start, "invalid number: missing integer part before decimal point")
			return token{tokenIllegal, start, l.text(start, l.offset)}
		}
		for isIdentChar(l.ch) {
			l.next()
		}
		return token{tokenSignedIdent, start, l.text(start, l.offset)}

	}

	l.errorf(l.offset, "unexpected character: %c (U+%04X)", l.ch, l.ch)
	l.next()
	return token{tokenIllegal, start, l.text(start, l.offset)}
}

func (l *lexer) readNumber() token {
	start := l.offset
	if isSign(l.ch) {
		l.next() // consume sign
	}

	base := 10
	hasDigits := false

	if l.ch == '0' {
		l.next()
		switch l.ch {
		case 'x':
			l.next()
			base = 16
		case 'o':
			l.next()
			base = 8
		case 'b':
			l.next()
			base = 2
		default:
			hasDigits = true
		}
	}

	for {
		switch {
		case l.ch == '_':
			if !hasDigits {
				l.errorf(l.offset, "separator '_' cannot appear at the start of a number")
			}
			l.next()
			continue
		case base == 16 && isHexDigit(l.ch):
			hasDigits = true
			l.next()
			continue
		case base != 16 && isDigit(l.ch) && l.ch < '0'+rune(base):
			hasDigits = true
			l.next()
			continue
		}
		// not a valid digit
		break
	}

	if !hasDigits {
		l.errorf(start, "invalid number: missing digits")
		return token{tokenIllegal, start, l.text(start, l.offset)}
	}

	switch base {
	case 2:
		return token{tokenBinary, start, l.text(start, l.offset)}
	case 8:
		return token{tokenOctal, start, l.text(start, l.offset)}
	case 16:
		return token{tokenHexadecimal, start, l.text(start, l.offset)}
	}

	if l.ch == '.' {
		// decimal point
		l.next()
		l.readDigits("fractional part")
	}

	if l.ch == 'e' || l.ch == 'E' {
		// exponent
		l.next()
		if isSign(l.ch) {
			l.next()
		}
		l.readDigits("exponent")
	}

	return token{tokenDecimal, start, l.text(start, l.offset)}
}

func (l *lexer) readDigits(part string) {
	hadDigits := false
loop:
	for {
		switch {
		case l.ch == '_':
			if !hadDigits {
				l.errorf(l.offset, "separator '_' cannot appear at the start of the %s", part)
			}
			l.next()
		case isDigit(l.ch):
			hadDigits = true
			l.next()
		default:
			// not a valid digit
			break loop
		}
	}
	if !hadDigits {
		l.errorf(l.offset, "invalid number: missing digits in %s", part)
	}
}

func (l *lexer) readNewline() token {
	start := l.offset
	ch := l.ch
	l.next()
	if ch == '\r' && l.ch == '\n' {
		l.next()
	}
	return token{tokenNewline, start, l.text(start, l.offset)}
}

func (l *lexer) lexMultiLineComment() token {
	switch ch := l.ch; ch {
	case runeEOF:
		l.errorf(l.offset, "unterminated multi-line comment")
		return token{tokenEOF, l.offset, ""}
	case '*':
		if l.peek() == '/' {
			start := l.offset
			l.next()
			l.next()
			l.popMode()
			return token{tokenMultiLineCommentEnd, start, "*/"}
		}
		l.next()
		return token{tokenMultiLineCommentContent, l.offset, "*"}
	case '/':
		if l.peek() == '*' {
			start := l.offset
			l.next()
			l.next()
			l.pushMode(modeMultiLineComment)
			return token{tokenMultiLineCommentStart, start, "/*"}
		}
		l.next()
		return token{tokenMultiLineCommentContent, l.offset, "/"}
	}

	start := l.offset
	for l.ch != runeEOF && l.ch != '*' && l.ch != '/' {
		l.next()
	}
	return token{tokenMultiLineCommentContent, start, l.text(start, l.offset)}
}
