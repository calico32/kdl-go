package kdl

import (
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/pkg/errors"
)

// Parse parses a KDL document from the provided reader and returns it.
func Parse(r io.Reader, opts ...ParseOption) (*Document, error) {
	return ParseNamed("<input>", r)
}

// ParseNamed is like [Parse], but allows specifying a name for the input
// source. Nodes and errors will reference this name in their locations.
func ParseNamed(name string, r io.Reader, opts ...ParseOption) (*Document, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	l := newLexer(name, src, nil)
	p := newParser(l, nil)
	for _, opt := range opts {
		opt(p)
	}
	return p.ParseDocument()
}

type ParseOption func(*parser)

func WithParseTrace(w io.Writer) ParseOption {
	return func(p *parser) {
		p.trace = w
	}
}

func WithLocations(v bool) ParseOption {
	return func(p *parser) {
		p.withLocations = v
	}
}

func newParser(lex *lexer, trace io.Writer) *parser {
	p := &parser{
		lexer:         lex,
		trace:         trace,
		withLocations: true,
	}
	lex.AddErrorHandler(func(pos Pos, err error) {
		p.errors = append(p.errors, errors.Wrapf(err, "lex error at %s", p.lexer.File().Location(pos)))
	})
	p.next()
	p.next()
	return p
}

type parser struct {
	lexer         *lexer
	token         token
	nextToken     token
	errors        []error
	trace         io.Writer
	withLocations bool
}

func (p *parser) errorf(pos Pos, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	err := errors.Errorf("parse error at %s: %s", p.lexer.File().Location(pos), msg)
	if p.trace != nil {
		_, _ = fmt.Fprintf(p.trace, "%+v\n", err)
	}
	p.errors = append(p.errors, err)
}

func (p *parser) errorExpected(expected string) {
	tok := p.token
	p.errorf(tok.Pos, "expected %s, got %s", expected, tok.Type)
}
func (p *parser) next() {
	p.token = p.nextToken
	tok := p.lexer.Next()
	p.nextToken = tok
}

func (p *parser) expect(tt tokenType) token {
	tok := p.token
	if tok.Type != tt {
		p.errorf(tok.Pos, "expected token type %v, got %v", tt, tok.Type)
	}
	p.next()
	return tok
}

// ParseDocument parses a KDL document and returns it.
//
//	document := bom? version? nodes
//	nodes := (line-space* node)* line-space*
func (p *parser) ParseDocument() (*Document, error) {
	d := &Document{}
	d.Nodes = p.parseNodes()
	p.expect(tokenEOF)
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	return d, nil
}

func (p *parser) parseNodes() (nodes []*Node) {
	p.skipLineSpace()
	for p.token.Type != tokenEOF && p.token.Type != tokenRBrace {
		node := p.parseNode(true)
		if node != nil {
			nodes = append(nodes, node)
		}
		switch p.token.Type {
		case tokenNewline, tokenSingleLineComment, tokenSemi, tokenEOF:
			p.next()
			p.skipLineSpace()
		case tokenRBrace:
			// OK, no terminator needed
		case tokenIllegal:
			// abort
			p.next() // make progress
			return nodes
		default:
			p.errorExpected("node terminator")
		}
	}
	return nodes
}

// readSlashdash reads a slashdash.
//
//	slashdash := '/-' line-space*
func (p *parser) readSlashdash() {
	p.expect(tokenSlashdash)
	p.skipLineSpace()
}

// parseNode parses a KDL node without the terminator and returns it.
//
//	node := base-node node-terminator
//	base-node := slashdash? type? node-space* string
//	  (node-space+ slashdash? node-prop-or-arg)*
//	  // slashdashed node-children must always be after props and args.
//	  (node-space+ slashdash node-children)*
//	  (node-space+ node-children)?
//	  (node-space+ slashdash node-children)*
//	  node-space*
//	node-prop-or-arg := prop | value
//	node-children := '{' nodes final-node? '}'
//	final-node := base-node node-terminator?
//	prop := string node-space* '=' node-space* value
//	value := type? node-space* (string | number | keyword)
//	type := '(' node-space* string node-space* ')'
//	node-terminator := single-line-comment | newline | ';' | eof
func (p *parser) parseNode(allowSlashdash bool) *Node {
	n := &Node{
		props: make(map[string]Value),
	}
	if allowSlashdash && p.token.Type == tokenSlashdash {
		p.readSlashdash()
		p.parseNode(false)
		return nil
	}
	if p.token.Type == tokenLParen {
		n.ty = p.parseType()
		n.typeValid = true
	}
	p.skipNodeSpace()
	if p.withLocations {
		n.loc = p.lexer.File().Location(p.token.Pos)
	}
	n.name = p.parseString()

	slashdashChildrenEncountered := false
	childrenEncountered := false
	for {
		switch p.token.Type {
		case tokenEOF, tokenNewline, tokenSingleLineComment, tokenSemi, tokenRBrace:
			// end of node
			return n
		case tokenIllegal:
			p.next() // make progress
			continue
		}

		if p.token.Type != tokenSlashdash && p.token.Type != tokenLBrace {
			ok := p.readNodeSpace()
			if !ok {
				p.next()
				continue
			}
			p.skipNodeSpace()
		}

		slashdash := false

		switch p.token.Type {
		case tokenEOF, tokenNewline, tokenSingleLineComment, tokenSemi, tokenRBrace:
			// end of node
			return n
		case tokenIllegal:
			p.next() // make progress
			continue
		}

		if p.token.Type == tokenSlashdash {
			p.readSlashdash()
			slashdash = true
		}

		if childrenEncountered && !slashdash {
			// once children have been encountered, no more non-slashdash
			// children are allowed; we stop parsing the node here
			return n
		}

		// children
		if p.token.Type == tokenLBrace {
			p.next() // consume LBrace
			nodes := p.parseNodes()
			p.expect(tokenRBrace)

			if !slashdash {
				childrenEncountered = true
				n.children.Nodes = nodes
			} else {
				slashdashChildrenEncountered = true
			}
			continue
		}

		// props or args
		if slashdashChildrenEncountered {
			// once slashdash children have been encountered, no more props
			// or args are allowed; we stop parsing the node here
			return n
		}

		switch p.token.Type {
		case tokenUnambiguousIdent, tokenSignedIdent, tokenDottedIdent,
			tokenQuotedString, tokenQuotedMultiLineString,
			tokenRawString, tokenRawMultiLineString:
			// string: could be prop name or arg value
			s := p.parseString()
			savepoint := p.savepoint()
			p.skipNodeSpace()
			if p.token.Type == tokenEqual {
				// prop
				p.next() // consume Equal
				p.skipNodeSpace()
				val := p.parseValue()
				if !slashdash {
					n.AddProperty(s, val)
				}
			} else {
				// arg
				p.restorepoint(savepoint)
				if !slashdash {
					n.args = append(n.args, NewString(s))
				}
			}

		default:
			if p.token.Type == tokenEOF {
				if slashdash {
					p.errorf(p.token.Pos, "expected value after slashdash")
					return n
				}
			}
			val := p.parseValue()
			if val == (Value{}) {
				p.next()
				continue
			}
			if !slashdash {
				n.args = append(n.args, val)
			}
		}
	}
}

type savepoint struct {
	token      token
	nextToken  token
	lexerState LexerState
}

func (p *parser) savepoint() savepoint {
	return savepoint{
		token:      p.token,
		nextToken:  p.nextToken,
		lexerState: p.lexer.Save(),
	}
}

func (p *parser) restorepoint(savepoint savepoint) {
	p.token = savepoint.token
	p.nextToken = savepoint.nextToken
	p.lexer.Restore(savepoint.lexerState)
}

// parseType parses a type annotation and returns the unquoted string value.
//
//	type := '(' node-space* string node-space* ')'
func (p *parser) parseType() string {
	p.expect(tokenLParen)
	p.skipNodeSpace()
	str := p.parseString()
	p.skipNodeSpace()
	p.expect(tokenRParen)
	return str
}

// parseString parses a string and returns its value.
//
//	string := identifier-string | quoted-string | raw-string
//	identifier-string := unambiguous-ident | signed-ident | dotted-ident
func (p *parser) parseString() string {
	tok := p.token
	switch tok.Type {
	case tokenUnambiguousIdent, tokenSignedIdent, tokenDottedIdent,
		tokenQuotedString, tokenQuotedMultiLineString,
		tokenRawString, tokenRawMultiLineString:
		p.next()
		return tok.Text
	default:
		p.errorExpected("string")
		return ""
	}
}

// parseValue parses a KDL string, number, or keyword and returns it.
//
//	number := keyword-number | hex | octal | binary | decimal
//	keyword := boolean | '#null'
//	keyword-number := '#inf' | '#-inf' | '#nan'
//	boolean := '#true' | '#false'
func (p *parser) parseValue() Value {
	var typeAnnot string
	var typeAnnotPresent bool
	if p.token.Type == tokenLParen {
		typeAnnot = p.parseType()
		typeAnnotPresent = true
		p.skipNodeSpace()
	}

	tok := p.token
	var value Value
	pos := p.token.Pos
	switch typ := tok.Type; typ {
	case tokenUnambiguousIdent, tokenSignedIdent, tokenDottedIdent,
		tokenQuotedString, tokenQuotedMultiLineString,
		tokenRawString, tokenRawMultiLineString:
		str := p.parseString()
		value = NewString(str)
	case tokenTrue:
		p.next()
		value = NewBool(true)
	case tokenFalse:
		p.next()
		value = NewBool(false)
	case tokenNull:
		p.next()
		value = NewNull()
	case tokenInf:
		p.next()
		value = infValue
	case tokenNegInf:
		p.next()
		value = negInfValue
	case tokenNaN:
		p.next()
		value = nanValue
	case tokenDecimal, tokenHexadecimal, tokenOctal, tokenBinary:
		value = p.parseNumber()
	default:
		p.errorExpected("value")
		return NewNull()
	}

	value = value.WithTypeAnnotation(typeAnnot, typeAnnotPresent)
	if p.withLocations {
		value = value.WithLocation(p.lexer.File().Location(pos))
	}
	return value
}

// parseNumber parses a KDL numeric literal and returns it.
func (p *parser) parseNumber() Value {
	digits := p.token.Text
	base := 10
	fp := false
	switch p.token.Type {
	case tokenDecimal:
		fp = strings.ContainsAny(digits, ".eE")
	case tokenHexadecimal:
		base = 16
		digits = digits[2:]
	case tokenOctal:
		base = 8
		digits = digits[2:]
	case tokenBinary:
		base = 2
		digits = digits[2:]
	default:
		p.errorExpected("number")
		return NewNull()
	}
	p.next()

	if fp {
		// floating point
		var f big.Float
		_, _, err := f.Parse(strings.ReplaceAll(digits, "_", ""), 10)
		if err != nil {
			p.errorf(p.token.Pos, "invalid float literal: %q", digits)
			return NewNull()
		}
		f64, prec := f.Float64()
		if prec == big.Exact {
			return NewFloat(f64)
		} else {
			return NewBigFloat(&f)
		}
	}

	// integer
	digits = strings.ReplaceAll(digits, "_", "")
	var i big.Int
	_, ok := i.SetString(digits, base)
	if !ok {
		p.errorf(p.token.Pos, "invalid integer literal: %q", digits)
		return NewNull()
	}
	if i.IsInt64() {
		return NewInt(int(i.Int64()))
	} else {
		return NewBigInt(&i)
	}
}
