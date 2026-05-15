package kdl

// TokenKind classifies a lexical token in the public token stream produced by
// [Tokenize]. Internal lexer token families (the various string, number, and
// comment sub-kinds) are collapsed into a single kind each.
type TokenKind int

const (
	// TokenEOF is the synthetic final token. Its Start/End point at the end of
	// the source.
	TokenEOF TokenKind = iota
	// TokenWhitespace is inline whitespace (spaces, tabs) but not newlines.
	TokenWhitespace
	// TokenNewline is a line break (LF or CRLF).
	TokenNewline
	// TokenLineContinuation is an escline backslash (`\`). The newline that
	// follows it is still emitted as a separate [TokenNewline].
	TokenLineContinuation
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenSemicolon
	TokenEqual
	// TokenIdent is a bare/unambiguous/signed/dotted identifier.
	TokenIdent
	// TokenString is any string literal (quoted/raw/single-line/multi-line).
	TokenString
	// TokenNumber is any numeric literal (decimal/hex/octal/binary).
	TokenNumber
	// TokenKeyword is a #-keyword value (#true, #false, #null, #inf, #-inf,
	// #nan) or, in v1, the unprefixed keywords true, false, and null.
	TokenKeyword
	// TokenComment is any comment span (single-line, or a piece of a
	// multi-line comment).
	TokenComment
	// TokenSlashdash is the `/-` slashdash prefix.
	TokenSlashdash
	// TokenIllegal is an unrecognized or malformed lexeme.
	TokenIllegal
)

func (k TokenKind) String() string {
	switch k {
	case TokenEOF:
		return "EOF"
	case TokenWhitespace:
		return "Whitespace"
	case TokenNewline:
		return "Newline"
	case TokenLineContinuation:
		return "LineContinuation"
	case TokenLBrace:
		return "LBrace"
	case TokenRBrace:
		return "RBrace"
	case TokenLParen:
		return "LParen"
	case TokenRParen:
		return "RParen"
	case TokenSemicolon:
		return "Semicolon"
	case TokenEqual:
		return "Equal"
	case TokenIdent:
		return "Ident"
	case TokenString:
		return "String"
	case TokenNumber:
		return "Number"
	case TokenKeyword:
		return "Keyword"
	case TokenComment:
		return "Comment"
	case TokenSlashdash:
		return "Slashdash"
	case TokenIllegal:
		return "Illegal"
	default:
		return "Unknown"
	}
}

// Token is a single lexical token with source locations. Start is inclusive,
// End is exclusive (one byte past the last byte of the token).
type Token struct {
	Kind  TokenKind
	Start Location
	End   Location
	Text  string
}

func tokenKindOf(t tokenType) TokenKind {
	switch t {
	case tokenEOF:
		return TokenEOF
	case tokenWS:
		return TokenWhitespace
	case tokenNewline:
		return TokenNewline
	case tokenBackslash:
		return TokenLineContinuation
	case tokenLBrace:
		return TokenLBrace
	case tokenRBrace:
		return TokenRBrace
	case tokenLParen:
		return TokenLParen
	case tokenRParen:
		return TokenRParen
	case tokenSemi:
		return TokenSemicolon
	case tokenEqual:
		return TokenEqual
	case tokenUnambiguousIdent, tokenSignedIdent, tokenDottedIdent:
		return TokenIdent
	case tokenQuotedString, tokenQuotedMultiLineString,
		tokenRawString, tokenRawMultiLineString:
		return TokenString
	case tokenHexadecimal, tokenOctal, tokenBinary, tokenDecimal:
		return TokenNumber
	case tokenInf, tokenNegInf, tokenNaN, tokenNull, tokenTrue, tokenFalse:
		return TokenKeyword
	case tokenSingleLineComment, tokenMultiLineCommentStart,
		tokenMultiLineCommentContent, tokenMultiLineCommentEnd:
		return TokenComment
	case tokenSlashdash:
		return TokenSlashdash
	default:
		return TokenIllegal
	}
}

// Tokenize lexes src into a complete token stream: whitespace, newlines,
// esclines (line continuations), and comments are all preserved, in source
// order. The stream always ends with a single [TokenEOF].
//
// Tokenize never returns an error. Malformed input yields [TokenIllegal] tokens
// rather than failing. It accepts the same options as [Parse]; only the version
// option is consulted.
func Tokenize(src []byte, opts ...ParseOption) []Token {
	version := VersionAuto
	for _, opt := range opts {
		if vo, ok := opt.(versionOption); ok {
			version = vo.v
			break
		}
	}

	l := newLexer("<input>", src, nil, version)
	f := l.File()

	// Bound the loop defensively: a well-behaved lexer always advances and
	// terminates at EOF, but a malformed-input bug should degrade to a
	// truncated stream rather than hang forever.
	maxIter := 4*len(src) + 16
	out := make([]Token, 0, len(src)/2+1)
	for range maxIter {
		t := l.Next()
		out = append(out, Token{
			Kind:  tokenKindOf(t.Type),
			Start: f.Location(t.Pos),
			End:   f.Location(t.EndPos),
			Text:  t.Text,
		})
		if t.Type == tokenEOF {
			return out
		}
	}
	return out
}
