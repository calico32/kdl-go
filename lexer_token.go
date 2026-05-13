package kdl

import (
	"strconv"
	"strings"
)

type tokenType uint8

const (
	tokenEOF tokenType = iota
	tokenIllegal
	tokenWS
	tokenBackslash
	tokenNewline

	tokenLBrace
	tokenRBrace
	tokenLParen
	tokenRParen
	tokenSemi
	tokenUnambiguousIdent // v2 unambiguous ident or v1 bare ident
	tokenSignedIdent
	tokenDottedIdent
	tokenQuotedString
	tokenQuotedMultiLineString
	tokenRawString
	tokenRawMultiLineString
	tokenEqual

	tokenHexadecimal
	tokenOctal
	tokenBinary
	tokenDecimal

	tokenInf
	tokenNegInf
	tokenNaN
	tokenNull  // v2 #null or v1 null
	tokenTrue  // v2 #true or v1 true
	tokenFalse // v2 #false or v1 false

	tokenSingleLineComment
	tokenMultiLineCommentStart
	tokenMultiLineCommentContent
	tokenMultiLineCommentEnd
	tokenSlashdash
	tokenVersion

	numTokens
)

var tokenTypeNames = map[tokenType]string{
	tokenEOF:                     "<eof>",
	tokenIllegal:                 "<illegal>",
	tokenWS:                      "<ws>",
	tokenBackslash:               "\\",
	tokenNewline:                 "<newline>",
	tokenLBrace:                  "{",
	tokenRBrace:                  "}",
	tokenLParen:                  "(",
	tokenRParen:                  ")",
	tokenSemi:                    ";",
	tokenUnambiguousIdent:        "<uident>",
	tokenSignedIdent:             "<sident>",
	tokenDottedIdent:             "<dident>",
	tokenQuotedString:            "<qstring>",
	tokenQuotedMultiLineString:   "<qmstring>",
	tokenRawString:               "<rstring>",
	tokenRawMultiLineString:      "<rmstring>",
	tokenEqual:                   "=",
	tokenHexadecimal:             "<hex>",
	tokenOctal:                   "<oct>",
	tokenBinary:                  "<bin>",
	tokenDecimal:                 "<dec>",
	tokenInf:                     "#inf",
	tokenNegInf:                  "#-inf",
	tokenNaN:                     "#nan",
	tokenNull:                    "#null",
	tokenTrue:                    "#true",
	tokenFalse:                   "#false",
	tokenSingleLineComment:       "<scomment>",
	tokenMultiLineCommentStart:   "<mcomment_start>",
	tokenMultiLineCommentContent: "<mcomment_content>",
	tokenMultiLineCommentEnd:     "<mcomment_end>",
	tokenSlashdash:               "/-",
	tokenVersion:                 "<version>",
}

func init() {
	missingNames := []tokenType{}
	for i := range numTokens {
		if _, ok := tokenTypeNames[i]; !ok {
			missingNames = append(missingNames, i)
		}
	}
	if len(missingNames) > 0 {
		var sb strings.Builder
		sb.WriteString("Missing token type names for:")
		for _, t := range missingNames {
			sb.WriteString(" ")
			sb.WriteString(strconv.Itoa(int(t)))
		}
		panic(sb.String())
	}
}

func (t tokenType) String() string {
	if name, ok := tokenTypeNames[t]; ok {
		return name
	}
	return "<unknown>"
}

type token struct {
	Type   tokenType
	Pos    Pos // start (inclusive)
	EndPos Pos // end (exclusive)
	Text   string
}
