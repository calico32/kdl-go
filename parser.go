package kdl

import (
	"fmt"
	"io"
	"math/big"
	"strings"
)

// Parse parses a KDL document from the provided reader and returns it.
func Parse(r io.Reader, opts ...ParseOption) (*Document, error) {
	return ParseNamed("<input>", r, opts...)
}

// ParseNamed is like [Parse], but allows specifying a name for the input
// source. Nodes and errors will reference this name in their locations.
func ParseNamed(name string, r io.Reader, opts ...ParseOption) (*Document, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	result, err := parseWithDiagnosticsFromBytes(name, src, opts...)
	if err != nil {
		return nil, err
	}
	if result.HasErrors() {
		for _, d := range result.Diagnostics {
			if d.Severity == SeverityError {
				return nil, fmt.Errorf("parse error at %s: %s", d.Start, d.Message)
			}
		}
	}
	return result.Document, nil
}

// ParseWithDiagnostics parses a KDL document and returns a [ParseResult]
// containing a (possibly partial) document and all diagnostics. Unlike [Parse],
// it never returns an error for parse problems — check [ParseResult.HasErrors]
// or inspect [ParseResult.Diagnostics] instead.
func ParseWithDiagnostics(r io.Reader, opts ...ParseOption) (*ParseResult, error) {
	return ParseNamedWithDiagnostics("<input>", r, opts...)
}

// ParseNamedWithDiagnostics is like [ParseWithDiagnostics] but lets you name
// the input source so that locations in diagnostics reference that name.
func ParseNamedWithDiagnostics(name string, r io.Reader, opts ...ParseOption) (*ParseResult, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseWithDiagnosticsFromBytes(name, src, opts...)
}

func parseWithDiagnosticsFromBytes(name string, src []byte, opts ...ParseOption) (*ParseResult, error) {
	version := VersionAuto
	for _, opt := range opts {
		if opt, ok := opt.(versionOption); ok {
			version = opt.v
			break
		}
	}
	l := newLexer(name, src, nil, version)
	p := newParser(l, nil, opts...)
	doc, diags := p.ParseDocument()

	hasErrors := func(ds []Diagnostic) bool {
		for _, d := range ds {
			if d.Severity == SeverityError {
				return true
			}
		}
		return false
	}

	if !hasErrors(diags) || p.version != VersionAuto {
		v := p.version
		if v == VersionAuto {
			v = Version2
		}
		return &ParseResult{Document: doc, Diagnostics: diags, Version: v}, nil
	}

	// try again as v1
	v1Opts := append(append([]ParseOption{}, opts...), WithVersion(Version1))
	l2 := newLexer(name, src, nil, Version1)
	p2 := newParser(l2, nil, v1Opts...)
	doc2, diags2 := p2.ParseDocument()
	if !hasErrors(diags2) {
		return &ParseResult{Document: doc2, Diagnostics: diags2, Version: Version1}, nil
	}
	if detectV1(string(src)) {
		return &ParseResult{Document: doc2, Diagnostics: diags2, Version: Version1}, nil
	}
	return &ParseResult{Document: doc, Diagnostics: diags, Version: Version2}, nil
}

type ParseOption interface {
	applyParser(*parser)
}

type parseOptionFunc func(*parser)

func (f parseOptionFunc) applyParser(p *parser) {
	f(p)
}

func WithParseTrace(w io.Writer) ParseOption {
	return parseOptionFunc(func(p *parser) {
		p.trace = w
	})
}

func WithLocations(v bool) ParseOption {
	return parseOptionFunc(func(p *parser) {
		p.withLocations = v
	})
}

func newParser(lex *lexer, trace io.Writer, options ...ParseOption) *parser {
	p := &parser{
		lexer:         lex,
		trace:         trace,
		withLocations: true,
		version:       VersionAuto,
	}
	lex.AddErrorHandler(func(pos Pos, err error) {
		loc := p.lexer.File().Location(pos)
		p.diagnostics = append(p.diagnostics, Diagnostic{
			Start:    loc,
			End:      loc,
			Severity: SeverityError,
			Message:  "lex error: " + err.Error(),
		})
	})
	for _, opt := range options {
		opt.applyParser(p)
	}
	p.lexer.version = p.version
	p.next()
	p.next()
	return p
}

type parser struct {
	lexer          *lexer
	token          token
	nextToken      token
	diagnostics    []Diagnostic
	parserErrCount int
	trace          io.Writer
	withLocations  bool
	version        Version
}

func (p *parser) errorf(pos Pos, format string, args ...any) {
	p.errorfRange(pos, p.token.EndPos, format, args...)
}

func (p *parser) errorfRange(startPos, endPos Pos, format string, args ...any) {
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Start:    p.lexer.File().Location(startPos),
		End:      p.lexer.File().Location(endPos),
		Severity: SeverityError,
		Message:  fmt.Sprintf(format, args...),
	})
	p.parserErrCount++
}

func (p *parser) errorExpected(expected string) {
	tok := p.token
	p.errorfRange(tok.Pos, tok.EndPos, "expected %s, got %s", expected, tok.Type)
}

func (p *parser) next() {
	p.token = p.nextToken
	tok := p.lexer.Next()
	if p.trace != nil {
		_, _ = fmt.Fprintf(p.trace, "lex %v: %q\n", tok.Type, tok.Text)
	}
	p.nextToken = tok
}

func (p *parser) expect(tt tokenType) token {
	tok := p.token
	if tok.Type != tt {
		p.errorfRange(tok.Pos, tok.EndPos, "expected token type %v, got %v", tt, tok.Type)
	}
	p.next()
	return tok
}

// ParseDocument parses a KDL document and returns it along with any diagnostics.
//
//	document := bom? version? nodes
//	nodes := (line-space* node)* line-space*
func (p *parser) ParseDocument() (*Document, []Diagnostic) {
	d := &Document{}
	d.Nodes, d.TrailingComments = p.parseNodes()
	p.expect(tokenEOF)
	return d, p.diagnostics
}

// detectV1 returns true if the input appears to be KDL v1 via heuristics.
// Lifted directly from kdl-rs (Apache 2.0 license):
// https://github.com/kdl-org/kdl-rs/blob/268f3a2d00d400877cc85530b85dbb145f0b2dfb/src/document.rs#L504-L519
func detectV1(input string) bool {
	if newline := strings.IndexRune(input, '\n'); newline != -1 {
		if strings.Contains(input[:newline], "kdl-version 1") {
			return true
		} else if strings.Contains(input[:newline], "kdl-version 2") {
			// explicit v2 declaration overrides any v1 heuristics
			return false
		}
	}
	if strings.Contains(input, " true") ||
		strings.Contains(input, " false") ||
		strings.Contains(input, " null") ||
		strings.Contains(input, "r#\"") ||
		strings.Contains(input, " \"\n") ||
		strings.Contains(input, " \"\r\n") {
		return true
	}

	return false
}

// detectV2 returns true if the input appears to be KDL v2 via heuristics.
// Lifted directly from kdl-rs (Apache 2.0 license):
// https://github.com/kdl-org/kdl-rs/blob/268f3a2d00d400877cc85530b85dbb145f0b2dfb/src/document.rs#L472-L502
// func detectV2(input string) bool {
// 	for line := range strings.SplitSeq(input, "\n") {
// 		if strings.Contains(line, "kdl-version 2") ||
// 			strings.Contains(line, "#true") ||
// 			strings.Contains(line, "#false") ||
// 			strings.Contains(line, "#null") ||
// 			strings.Contains(line, "#inf") ||
// 			strings.Contains(line, "#-inf") ||
// 			strings.Contains(line, "#nan") ||
// 			strings.Contains(line, " #\"") ||
// 			strings.Contains(line, "\"\"\"") {
// 			return true
// 		}
// 		if !strings.Contains(line, "\"") {
// 			fields := strings.Fields(line)[1:]
// 			for _, field := range fields {
// 				if len(field) > 0 {
// 					if !isSign(rune(field[0])) && !isDigit(rune(field[0])) {
// 						return true
// 					}
// 				}
// 			}
// 		}
// 	}

// 	return false
// }

// syncToNodeBoundary skips tokens until a node boundary without consuming it.
// On return, the current token is always newline, semi, }, or EOF.
func (p *parser) syncToNodeBoundary() {
	for {
		switch p.token.Type {
		case tokenEOF, tokenNewline, tokenSemi, tokenRBrace:
			return
		}
		p.next()
	}
}

func (p *parser) parseNodes() (nodes []*Node, trailing []Comment) {
	// collect comments and blank lines before the first node
	pendingComments, pendingBlankLine := p.collectBetweenNodes(0)

	for p.token.Type != tokenEOF && p.token.Type != tokenRBrace {
		// top-level slashdash-commented node
		if p.token.Type == tokenSlashdash {
			slashStart := p.token.Pos
			slashEnd := p.token.EndPos
			p.next() // consume /-
			p.skipLineSpace()
			slashedNode := p.parseNode()
			if slashedNode != nil {
				c := Comment{
					kind:            CommentSlashdash,
					node:            slashedNode,
					blankLineBefore: pendingBlankLine,
				}
				if p.withLocations {
					c.start = p.lexer.File().Location(slashStart)
					c.end = p.lexer.File().Location(slashEnd)
				}
				pendingComments = append(pendingComments, c)
				pendingBlankLine = false
			}
			// consume terminator
			switch p.token.Type {
			case tokenSingleLineComment:
				if slashedNode != nil {
					c := Comment{kind: CommentSingleLine, text: p.token.Text}
					if p.withLocations {
						c.start = p.lexer.File().Location(p.token.Pos)
						c.end = p.lexer.File().Location(p.token.EndPos)
					}
					slashedNode.trailingComment = &c
				}
				p.next()
				more, blank := p.collectBetweenNodes(1)
				pendingComments = append(pendingComments, more...)
				if blank {
					pendingBlankLine = true
				}
			case tokenNewline, tokenSemi:
				wasNewline := p.token.Type == tokenNewline
				p.next()
				start := 0
				if wasNewline {
					start = 1
				}
				more, blank := p.collectBetweenNodes(start)
				pendingComments = append(pendingComments, more...)
				if blank {
					pendingBlankLine = true
				}
			case tokenEOF, tokenRBrace:
				// nothing to consume — outer loop will exit
			default:
				p.errorExpected("node terminator")
				p.syncToNodeBoundary()
			}
			continue
		}

		node := p.parseNode()
		if node != nil {
			node.leadingComments = pendingComments
			node.blankLineBefore = pendingBlankLine
			pendingComments = nil
			pendingBlankLine = false
			nodes = append(nodes, node)
		}

		switch p.token.Type {
		case tokenSingleLineComment:
			if node != nil {
				c := Comment{kind: CommentSingleLine, text: p.token.Text}
				if p.withLocations {
					c.start = p.lexer.File().Location(p.token.Pos)
					c.end = p.lexer.File().Location(p.token.EndPos)
				}
				node.trailingComment = &c
			}
			p.next()
			// single-line comment token already includes its newline, so pass
			// initialNewlines=1 to correctly detect a blank line after it
			pendingComments, pendingBlankLine = p.collectBetweenNodes(1)
		case tokenNewline, tokenSemi:
			wasNewline := p.token.Type == tokenNewline
			p.next()
			start := 0
			if wasNewline {
				start = 1
			}
			pendingComments, pendingBlankLine = p.collectBetweenNodes(start)
		case tokenEOF, tokenRBrace:
			// no terminator needed
		default:
			// leftover token after node (or after failed recovery). record
			// error, sync to next boundary, then consume it to make progress
			p.errorExpected("node terminator")
			p.syncToNodeBoundary()
			wasNewline := p.token.Type == tokenNewline
			if wasNewline || p.token.Type == tokenSemi {
				p.next()
			}
			start := 0
			if wasNewline {
				start = 1
			}
			pendingComments, pendingBlankLine = p.collectBetweenNodes(start)
		}
	}

	trailing = pendingComments
	return
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
func (p *parser) parseNode() (n *Node) {
	n = &Node{
		props: make(map[string]Value),
	}
	var lastEndPos Pos
	if p.withLocations {
		defer func() {
			if n != nil && lastEndPos > 0 {
				n.endLoc = p.lexer.File().Location(lastEndPos)
			}
		}()
	}
	if p.token.Type == tokenLParen {
		savedParserErrs := p.parserErrCount
		var contentStart, contentEnd Pos
		n.typ, contentStart, contentEnd = p.parseTypeRange()
		n.typeValid = true
		if p.withLocations {
			n.typeAnnotStart = p.lexer.File().Location(contentStart)
			n.typeAnnotEnd = p.lexer.File().Location(contentEnd)
		}
		if p.parserErrCount > savedParserErrs {
			p.syncToNodeBoundary()
			return nil
		}
	}
	p.skipNodeSpace()
	lastEndPos = p.token.EndPos
	if p.withLocations {
		n.loc = p.lexer.File().Location(p.token.Pos)
		n.nameEndLoc = p.lexer.File().Location(p.token.EndPos)
	}
	savedParserErrs := p.parserErrCount
	n.name = p.parseString()
	if p.parserErrCount > savedParserErrs {
		p.syncToNodeBoundary()
		return nil
	}

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
		var slashStart, slashEnd Pos

		switch p.token.Type {
		case tokenEOF, tokenNewline, tokenSingleLineComment, tokenSemi, tokenRBrace:
			// end of node
			return n
		case tokenIllegal:
			p.next() // make progress
			continue
		}

		if p.token.Type == tokenSlashdash {
			slashStart = p.token.Pos
			slashEnd = p.token.EndPos
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
			// detect whether the children block was inline in source:
			// inline: { child1; child2 } — first token after { is not a newline
			// multiline: {\n  child\n} — first token after { is a newline
			wasInline := p.token.Type != tokenNewline
			nodes, childTrailing := p.parseNodes()

			if p.token.Type == tokenRBrace {
				lastEndPos = p.token.EndPos
				p.next()
			} else {
				// missing closing brace — record error, store what we have, return
				p.errorfRange(p.token.Pos, p.token.EndPos, "expected token type }, got %v", p.token.Type)
				if !slashdash {
					childrenEncountered = true
					n.children.Nodes = nodes
					n.children.TrailingComments = childTrailing
					n.childrenInline = &wasInline
				} else {
					slashdashChildrenEncountered = true
					sd := InlineSlashdash{
						kind:     InlineSlashdashChildren,
						children: Document{Nodes: nodes, TrailingComments: childTrailing},
					}
					sd.childrenInline = &wasInline
					if p.withLocations {
						sd.slashdashStart = p.lexer.File().Location(slashStart)
						sd.slashdashEnd = p.lexer.File().Location(slashEnd)
					}
					n.inlineSlashdashes = append(n.inlineSlashdashes, sd)
				}
				return n
			}

			if !slashdash {
				childrenEncountered = true
				n.children.Nodes = nodes
				n.children.TrailingComments = childTrailing
				n.childrenInline = &wasInline
			} else {
				slashdashChildrenEncountered = true
				sd := InlineSlashdash{
					kind:     InlineSlashdashChildren,
					children: Document{Nodes: nodes, TrailingComments: childTrailing},
				}
				sd.childrenInline = &wasInline
				if p.withLocations {
					sd.slashdashStart = p.lexer.File().Location(slashStart)
					sd.slashdashEnd = p.lexer.File().Location(slashEnd)
				}
				n.inlineSlashdashes = append(n.inlineSlashdashes, sd)
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
			typ := p.token.Type
			keyStart := p.token.Pos
			keyEnd := p.token.EndPos
			s := p.parseString()
			savepoint := p.savepoint()
			p.skipNodeSpace()
			if p.token.Type == tokenEqual {
				// prop
				p.next() // consume Equal
				p.skipNodeSpace()
				savedParserErrs := p.parserErrCount
				val := p.parseValue()
				if p.parserErrCount > savedParserErrs {
					p.syncToNodeBoundary()
					return n
				}
				lastEndPos = val.endLocation.Offset
				if !slashdash {
					n.AddProperty(s, val)
					if p.withLocations {
						n.setPropertyKeyLocation(s,
							p.lexer.File().Location(keyStart),
							p.lexer.File().Location(keyEnd))
					}
				} else {
					sd := InlineSlashdash{
						kind:           InlineSlashdashProp,
						afterPropCount: len(n.propOrder),
						propKey:        s,
						propVal:        val,
					}
					if p.withLocations {
						sd.slashdashStart = p.lexer.File().Location(slashStart)
						sd.slashdashEnd = p.lexer.File().Location(slashEnd)
						sd.propKeyStart = p.lexer.File().Location(keyStart)
						sd.propKeyEnd = p.lexer.File().Location(keyEnd)
					}
					n.inlineSlashdashes = append(n.inlineSlashdashes, sd)
				}
			} else {
				// arg
				p.restorepoint(savepoint)
				lastEndPos = keyEnd
				if !slashdash {
					if p.version == Version1 && typ == tokenUnambiguousIdent {
						p.errorf(p.token.Pos, "unexpected identifier %s (must be quoted)", s)
					}
					arg := NewString(s)
					if p.withLocations {
						arg = arg.WithLocation(p.lexer.File().Location(keyStart))
						arg = arg.WithEndLocation(p.lexer.File().Location(keyEnd))
					}
					n.args = append(n.args, arg)
					n.entries = append(n.entries, nodeEntryArg)
				} else {
					arg := NewString(s)
					if p.withLocations {
						arg = arg.WithLocation(p.lexer.File().Location(keyStart))
						arg = arg.WithEndLocation(p.lexer.File().Location(keyEnd))
					}
					sd := InlineSlashdash{
						kind:          InlineSlashdashArg,
						afterArgCount: len(n.args),
						argValue:      arg,
					}
					if p.withLocations {
						sd.slashdashStart = p.lexer.File().Location(slashStart)
						sd.slashdashEnd = p.lexer.File().Location(slashEnd)
					}
					n.inlineSlashdashes = append(n.inlineSlashdashes, sd)
				}
			}

		default:
			if p.token.Type == tokenEOF {
				if slashdash {
					p.errorfRange(p.token.Pos, p.token.EndPos, "expected value after slashdash")
					return n
				}
				return n
			}
			savedParserErrs := p.parserErrCount
			val := p.parseValue()
			if p.parserErrCount > savedParserErrs {
				p.syncToNodeBoundary()
				return n
			}
			lastEndPos = val.endLocation.Offset
			if !slashdash {
				n.args = append(n.args, val)
				n.entries = append(n.entries, nodeEntryArg)
			} else {
				sd := InlineSlashdash{
					kind:          InlineSlashdashArg,
					afterArgCount: len(n.args),
					argValue:      val,
				}
				if p.withLocations {
					sd.slashdashStart = p.lexer.File().Location(slashStart)
					sd.slashdashEnd = p.lexer.File().Location(slashEnd)
				}
				n.inlineSlashdashes = append(n.inlineSlashdashes, sd)
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

// parseTypeRange parses a type annotation and also returns the byte positions
// of the content token (the identifier, not the surrounding parens).
// contentStart is inclusive, contentEnd is exclusive.
func (p *parser) parseTypeRange() (s string, contentStart, contentEnd Pos) {
	p.expect(tokenLParen)
	p.skipNodeSpace()
	contentStart = p.token.Pos
	contentEnd = p.token.EndPos
	s = p.parseString()
	p.skipNodeSpace()
	p.expect(tokenRParen)
	return
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
	var typeAnnotContentStart, typeAnnotContentEnd Pos
	if p.token.Type == tokenLParen {
		typeAnnot, typeAnnotContentStart, typeAnnotContentEnd = p.parseTypeRange()
		typeAnnotPresent = true
		p.skipNodeSpace()
	}

	tok := p.token
	var value Value
	pos := p.token.Pos
	switch tok.Type {
	case tokenUnambiguousIdent:
		// only valid as a value in v2
		if p.version == Version1 {
			p.errorf(tok.Pos, "unexpected identifier (must be quoted)")
		}
		fallthrough
	case tokenSignedIdent, tokenDottedIdent,
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
		value.numericLiteral = tok.Text
	case tokenNegInf:
		p.next()
		value = negInfValue
		value.numericLiteral = tok.Text
	case tokenNaN:
		p.next()
		value = nanValue
		value.numericLiteral = tok.Text
	case tokenDecimal, tokenHexadecimal, tokenOctal, tokenBinary:
		value = p.parseNumber()
	default:
		p.errorExpected("value")
		return NewNull()
	}

	value = value.WithTypeAnnotation(typeAnnot, typeAnnotPresent)
	if p.withLocations {
		value = value.WithLocation(p.lexer.File().Location(pos))
		value = value.WithEndLocation(p.lexer.File().Location(tok.EndPos))
		if typeAnnotPresent {
			value = value.WithTypeAnnotationRange(
				p.lexer.File().Location(typeAnnotContentStart),
				p.lexer.File().Location(typeAnnotContentEnd),
			)
		}
	}
	return value
}

// parseNumber parses a KDL numeric literal and returns it.
func (p *parser) parseNumber() Value {
	literal := p.token.Text
	digits := literal
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
			return NewFloat(f64).WithNumericLiteral(literal)
		} else {
			return NewBigFloat(&f).WithNumericLiteral(literal)
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
		return NewInt(int(i.Int64())).WithNumericLiteral(literal)
	} else {
		return NewBigInt(&i).WithNumericLiteral(literal)
	}
}
