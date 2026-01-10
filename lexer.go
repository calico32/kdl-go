package kdl

import (
	"fmt"
	"io"

	"unicode/utf8"
	"unsafe"

	"github.com/pkg/errors"
)

type LexerState struct {
	offset     Pos
	readOffset Pos
	ch         rune
	modeStack  []lexerMode
}

type lexer struct {
	file          *file
	offset        Pos
	readOffset    Pos
	ch            rune
	modeStack     []lexerMode
	errors        []error
	errorHandlers []func(Pos, error)
	trace         io.Writer
	version       Version
}

type lexerMode int

const (
	modeDefault lexerMode = iota
	modeMultiLineComment
)

const (
	runeEOF = 0
	runeBOM = 0xfeff
)

func newLexer(name string, src []byte, trace io.Writer, version Version) *lexer {
	l := &lexer{
		file:    newFile(name, src),
		trace:   trace,
		version: version,
	}
	l.next()
	if l.ch == runeBOM {
		// skip BOM
		l.next()
	}
	return l
}

func (l *lexer) Save() LexerState {
	return LexerState{
		offset:     l.offset,
		readOffset: l.readOffset,
		ch:         l.ch,
		modeStack:  append([]lexerMode{}, l.modeStack...),
	}
}

func (l *lexer) Restore(state LexerState) {
	l.offset = state.offset
	l.readOffset = state.readOffset
	l.ch = state.ch
	l.modeStack = append([]lexerMode{}, state.modeStack...)
}

func (l *lexer) errorf(offset Pos, format string, args ...any) {
	err := errors.Errorf(format, args...)
	l.errors = append(l.errors, errors.Wrapf(err, "lex error at %s", l.file.Location(offset)))
	for _, handler := range l.errorHandlers {
		handler(offset, err)
	}
}

func (l *lexer) AddErrorHandler(fn func(Pos, error)) {
	l.errorHandlers = append(l.errorHandlers, fn)
}

func (l *lexer) File() *file {
	return l.file
}

func (l *lexer) next() {
	if l.readOffset >= Pos(len(l.file.src)) {
		l.offset = Pos(len(l.file.src))
		l.ch = runeEOF
		return
	}

	l.offset = l.readOffset

	nextRune := rune(l.file.src[l.readOffset])

	if nextRune < utf8.RuneSelf {
		// ASCII, single byte
		l.readOffset++
		l.ch = nextRune
	} else {
		// not ASCII, multi-byte
		ch, width := utf8.DecodeRune(l.file.src[l.readOffset:])
		if ch == utf8.RuneError {
			l.errorf(l.offset, "invalid UTF-8 encoding")
		}
		l.readOffset += Pos(width)
		l.ch = ch
	}

	// 3.19. Disallowed Literal Code Points
	// https://kdl.dev/spec/#section-3.19
	switch {
	case isDisallowedChar(l.ch): // bidi control characters
		l.errorf(l.offset, "invalid control character: U+%04X", l.ch)
	case l.ch == runeBOM:
		if l.offset != 0 {
			l.errorf(l.offset, "unexpected BOM character")
		}
	}
}

func (l *lexer) peek() rune {
	if l.readOffset >= Pos(len(l.file.src)) {
		return 0
	}

	ch, _ := utf8.DecodeRune(l.file.src[l.readOffset:])
	return ch
}

func (l *lexer) match(str string) bool {
	pos := l.offset
	for i := 0; i < len(str); i++ {
		if pos >= Pos(len(l.file.src)) || l.file.src[pos] != str[i] {
			return false
		}
		pos++
	}
	return true
}

func (l *lexer) pushMode(mode lexerMode) {
	l.modeStack = append(l.modeStack, mode)
}

func (l *lexer) popMode() {
	if len(l.modeStack) == 0 {
		panic("lexer mode stack underflow")
	}
	l.modeStack = l.modeStack[:len(l.modeStack)-1]
}

// text returns the substring of the source from start to end positions. It
// avoids making a copy of the source data.
func (l *lexer) text(start Pos, end Pos) string {
	sl := unsafe.Pointer(unsafe.SliceData(l.file.src))
	ptr := unsafe.Add(sl, start)
	return unsafe.String((*byte)(ptr), end-start)
}

func (l *lexer) currentMode() lexerMode {
	if len(l.modeStack) == 0 {
		return modeDefault
	}
	return l.modeStack[len(l.modeStack)-1]
}

func (l *lexer) Next() (t token) {
	defer func() {
		if l.trace != nil {
			_, _ = fmt.Fprintf(l.trace, "lex %v: %q\n", t.Type, t.Text)
		}
	}()
	switch l.currentMode() {
	case modeDefault:
		return l.lexDefault()
	case modeMultiLineComment:
		return l.lexMultiLineComment()
	default:
		panic("unhandled lexer mode")
	}
}
