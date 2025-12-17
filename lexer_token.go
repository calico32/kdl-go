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
	tokenUnambiguousIdent
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

	tokenNull
	tokenInf
	tokenNegInf
	tokenNaN
	tokenTrue
	tokenFalse

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
	tokenNull:                    "null",
	tokenEqual:                   "=",
	tokenHexadecimal:             "<hex>",
	tokenOctal:                   "<oct>",
	tokenBinary:                  "<bin>",
	tokenDecimal:                 "<dec>",
	tokenInf:                     "#inf",
	tokenNegInf:                  "#-inf",
	tokenNaN:                     "#nan",
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
	for i := tokenType(0); i < numTokens; i++ {
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
	Type tokenType
	Pos  Pos
	Text string
}
