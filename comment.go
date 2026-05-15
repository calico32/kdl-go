package kdl

// CommentKind is a kind of KDL comment.
type CommentKind int

const (
	// CommentSingleLine is a // ... comment. Its Text includes the trailing newline.
	CommentSingleLine CommentKind = iota
	// CommentMultiLine is a /* ... */ comment. Its Text is the full literal text without
	// a trailing newline.
	CommentMultiLine
	// CommentSlashdash is a /- node comment. Its SlashedNode holds the commented-out
	// node (already parsed); the formatter re-formats it as KDL.
	CommentSlashdash
)

// A Comment is a KDL single-line (//), multi-line (/* */), or slashdash (/-)
// comment.
type Comment struct {
	kind            CommentKind
	text            string // raw text for CommentSingleLine / CommentMultiLine
	node            *Node  // parsed node for CommentSlashdash
	start           Location
	end             Location
	blankLineBefore bool
}

func (c Comment) Kind() CommentKind { return c.kind }

// Text returns the raw source text of a single-line or multi-line comment,
// preserved exactly as as written (following the spec rules for what is
// included in the text). If the comment is a slashdash comment, Text returns
// "".
func (c Comment) Text() string { return c.text }

// SlashedNode returns the commented-out node for a slashdash comment. If the
// comment is not a slashdash comment, SlashedNode returns nil.
func (c Comment) SlashedNode() *Node { return c.node }

// Start returns the source location of the first character of the comment token
// (the leading / of //, /*, or /-). If this comment was created without
// location tracking, Start returns a zero Location.
func (c Comment) Start() Location { return c.start }

// End returns the exclusive end location of the comment token. For single-line
// comments the end is immediately after the trailing newline; for multi-line
// comments it is after */; for slashdash it is after /-. If this comment was
// created without location tracking, End returns a zero Location.
func (c Comment) End() Location { return c.end }

// InlineSlashdashKind identifies the target of an inline /- comment within a node body.
type InlineSlashdashKind int

const (
	// InlineSlashdashArg is a /- before an argument value inside a node.
	InlineSlashdashArg InlineSlashdashKind = iota
	// InlineSlashdashProp is a /- before a key=value property inside a node.
	InlineSlashdashProp
	// InlineSlashdashChildren is a /- before a children block { ... } inside a node.
	InlineSlashdashChildren
)

// An InlineSlashdash is a KDL slashdash (/-) comment on an argument, property,
// or children block within a node body.
type InlineSlashdash struct {
	kind InlineSlashdashKind

	// InlineSlashdashArg:

	afterArgCount int   // number of real arguments before this slashdash in source
	argValue      Value // value of commented-out argument

	// InlineSlashdashProp:

	afterPropCount int    // number of real properties before this in source
	propKey        string // key of commented-out property
	propVal        Value  // value of commented-out property
	propKeyStart   Location
	propKeyEnd     Location

	// InlineSlashdashChildren:

	children       Document // children of commented-out block
	childrenInline *bool

	slashdashStart Location // position of /-
	slashdashEnd   Location // end position of /-
}

func (s InlineSlashdash) Kind() InlineSlashdashKind { return s.kind }
func (s InlineSlashdash) AfterArgCount() int        { return s.afterArgCount }
func (s InlineSlashdash) ArgValue() Value           { return s.argValue }
func (s InlineSlashdash) AfterPropCount() int       { return s.afterPropCount }
func (s InlineSlashdash) PropKey() string           { return s.propKey }
func (s InlineSlashdash) PropVal() Value            { return s.propVal }
func (s InlineSlashdash) Children() *Document       { return &s.children }
func (s InlineSlashdash) ChildrenInline() (inline, ok bool) {
	if s.childrenInline == nil {
		return false, false
	}
	return *s.childrenInline, true
}

// SlashdashStart returns the source location of the / in /-. If location
// tracking is off, returns a zero Location.
func (s InlineSlashdash) SlashdashStart() Location { return s.slashdashStart }

// SlashdashEnd returns the exclusive source location after /-. If location
// tracking is off, returns a zero Location.
func (s InlineSlashdash) SlashdashEnd() Location { return s.slashdashEnd }

// PropKeyLocation returns the source range of the property key token for an
// InlineSlashdashProp comment. If this is not an InlineSlashdashProp or if
// location tracking is off, returns ok=false.
func (s InlineSlashdash) PropKeyLocation() (start, end Location, ok bool) {
	if s.kind != InlineSlashdashProp || s.propKeyStart.Line == 0 {
		return
	}
	return s.propKeyStart, s.propKeyEnd, true
}
