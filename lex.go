package kdl

func (l *lexer) lexDefault() token {
	start := l.offset
	switch ch := l.ch; {
	case ch == runeEOF:
		return l.tok(tokenEOF, l.offset, "")

	case l.version != Version1 && isIdentStartChar(ch):
		// v2 unambiguous ident
		l.next()
		for isIdentChar(l.ch) {
			l.next()
		}
		lit := l.text(start, l.offset)
		switch lit {
		case "true", "false", "null", "inf", "-inf", "nan":
			l.errorf(start, "invalid identifier: %q (prefix with '#' for keyword or quote for string)", lit)
		}
		return l.tok(tokenUnambiguousIdent, start, lit)

	case l.version == Version1 && ch == 'r' && (l.peek() == '"' || l.peek() == '#'):
		// v1 raw string
		return l.readV1RawString()

	case l.version == Version1 && (isV1IdentStartChar(ch, false) || (!isDigit(l.peek()) && isV1IdentStartChar(ch, true))):
		// v1 bare ident
		l.next()
		for isV1IdentChar(l.ch) {
			l.next()
		}
		lit := l.text(start, l.offset)
		switch lit {
		case "true":
			return l.tok(tokenTrue, start, lit)
		case "false":
			return l.tok(tokenFalse, start, lit)
		case "null":
			return l.tok(tokenNull, start, lit)
		}
		return l.tok(tokenUnambiguousIdent, start, lit)

	case isUnicodeSpace(ch):
		l.next()
		for isUnicodeSpace(l.ch) {
			l.next()
		}
		return l.tok(tokenWS, start, l.text(start, l.offset))

	case isNewline(ch):
		return l.readNewline()

	case ch == '/':
		l.next()
		if l.ch == '*' {
			l.next()
			l.pushMode(modeMultiLineComment)
			return l.tok(tokenMultiLineCommentStart, start, "/*")
		}
		if l.ch == '/' {
			l.next()
			for l.ch != runeEOF && !isNewline(l.ch) {
				l.next()
			}
			if isNewline(l.ch) {
				l.next()
			}
			return l.tok(tokenSingleLineComment, start, l.text(start, l.offset))
		}
		if l.ch == '-' {
			l.next()
			return l.tok(tokenSlashdash, start, "/-")
		}

	case ch == '\\':
		l.next()
		return l.tok(tokenBackslash, start, "\\")

	case ch == '{':
		l.next()
		return l.tok(tokenLBrace, start, "{")

	case ch == '}':
		l.next()
		return l.tok(tokenRBrace, start, "}")

	case ch == '(':
		l.next()
		return l.tok(tokenLParen, start, "(")

	case ch == ')':
		l.next()
		return l.tok(tokenRParen, start, ")")

	case ch == ';':
		l.next()
		return l.tok(tokenSemi, start, ";")

	case ch == '=':
		l.next()
		return l.tok(tokenEqual, start, "=")

	case l.version != Version1 && ch == '#':
		// v2 keyword or v2 raw string
		l.next()
		if l.ch == '-' || isLetter(l.ch) {
			l.next()
			for isLetter(l.ch) {
				l.next()
			}
			lit := l.text(start, l.offset)
			switch lit {
			case "#inf":
				return l.tok(tokenInf, start, lit)
			case "#-inf":
				return l.tok(tokenNegInf, start, lit)
			case "#nan":
				return l.tok(tokenNaN, start, lit)
			case "#true":
				return l.tok(tokenTrue, start, lit)
			case "#false":
				return l.tok(tokenFalse, start, lit)
			case "#null":
				return l.tok(tokenNull, start, lit)
			default:
				l.errorf(start, "invalid keyword: %q", lit)
				return l.tok(tokenIllegal, start, lit)
			}
		}

		return l.readRawString()

	case ch == '"':
		return l.readQuotedString()

	case isSign(ch) && isDigit(l.peek()):
		return l.readNumber()

	case isDigit(ch):
		return l.readNumber()

	case l.version != Version1 && isSign(ch) && l.peek() == '.':
		// v2 dotted-ident
		// dotted-ident := sign? '.' ((identifier-char - digit) identifier-char*)?
		l.next() // consume sign
		fallthrough
	case l.version != Version1 && ch == '.':
		// v2 dotted-ident
		l.next() // consume '.'
		if isDigit(l.ch) {
			// invalid number - integer part is missing
			l.errorf(start, "invalid number: missing integer part before decimal point")
			return l.tok(tokenIllegal, start, l.text(start, l.offset))
		}
		for isIdentChar(l.ch) {
			l.next()
		}
		return l.tok(tokenDottedIdent, start, l.text(start, l.offset))

	case l.version != Version1 && isSign(ch):
		// v2 signed-ident
		// signed-ident := sign ((identifier-char - digit - '.') identifier-char*)?
		l.next() // consume sign
		if isDigit(l.ch) {
			// invalid number - integer part is missing
			l.errorf(start, "invalid number: missing integer part before decimal point")
			return l.tok(tokenIllegal, start, l.text(start, l.offset))
		}
		for isIdentChar(l.ch) {
			l.next()
		}
		return l.tok(tokenSignedIdent, start, l.text(start, l.offset))

	}

	l.errorf(l.offset, "unexpected character: %c (U+%04X)", l.ch, l.ch)
	l.next()
	return l.tok(tokenIllegal, start, l.text(start, l.offset))
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
		return l.tok(tokenIllegal, start, l.text(start, l.offset))
	}

	switch base {
	case 2:
		return l.tok(tokenBinary, start, l.text(start, l.offset))
	case 8:
		return l.tok(tokenOctal, start, l.text(start, l.offset))
	case 16:
		return l.tok(tokenHexadecimal, start, l.text(start, l.offset))
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

	return l.tok(tokenDecimal, start, l.text(start, l.offset))
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
	return l.tok(tokenNewline, start, l.text(start, l.offset))
}

func (l *lexer) lexMultiLineComment() token {
	switch ch := l.ch; ch {
	case runeEOF:
		l.errorf(l.offset, "unterminated multi-line comment")
		return l.tok(tokenEOF, l.offset, "")
	case '*':
		if l.peek() == '/' {
			start := l.offset
			l.next()
			l.next()
			l.popMode()
			return l.tok(tokenMultiLineCommentEnd, start, "*/")
		}
		start := l.offset
		l.next()
		return l.tok(tokenMultiLineCommentContent, start, "*")
	case '/':
		if l.peek() == '*' {
			start := l.offset
			l.next()
			l.next()
			l.pushMode(modeMultiLineComment)
			return l.tok(tokenMultiLineCommentStart, start, "/*")
		}
		start := l.offset
		l.next()
		return l.tok(tokenMultiLineCommentContent, start, "/")
	}

	start := l.offset
	for l.ch != runeEOF && l.ch != '*' && l.ch != '/' {
		l.next()
	}
	return l.tok(tokenMultiLineCommentContent, start, l.text(start, l.offset))
}
