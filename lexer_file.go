package kdl

import "fmt"

type file struct {
	name string
	// src is the complete source text of the KDL document. It MUST NOT be
	// modified after the File is created; immutable strings may point into it.
	src   []byte
	lines []int
}

func newFile(name string, _src []byte) *file {
	src := make([]byte, len(_src))
	copy(src, _src)
	f := &file{
		name: name,
		src:  src,
	}

	var lines []int
	line := 0
	for offset, b := range src {
		if line >= 0 {
			lines = append(lines, line)
		}
		line = -1
		if b == '\n' {
			line = offset + 1
		}
	}

	f.lines = lines
	return f
}

type Pos int

type Location struct {
	Offset   Pos    // byte offset in the file
	Filename string // optional filename
	Line     int    // 1-based line number
	Column   int    // 1-based column number (in bytes)
}

// Location returns the [Location] (line and column) corresponding to pos. If
// pos is out of bounds, Location returns a zero Location.
func (f *file) Location(pos Pos) (loc Location) {
	if i := searchInts(f.lines, int(pos)); i >= 0 {
		loc.Offset = pos
		loc.Filename = f.name
		loc.Line, loc.Column = i+1, int(pos)-f.lines[i]+1
	}
	return
}

func searchInts(a []int, x int) int {
	// This function body is a manually inlined version of:
	// return sort.Search(len(a), func(i int) bool { return a[i] > x }) - 1

	i, j := 0, len(a)
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if a[h] <= x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i - 1
}

func (l Location) String() string {
	if l.Filename != "" {
		return fmt.Sprintf("%s:%d:%d", l.Filename, l.Line, l.Column)
	}
	return fmt.Sprintf("%d:%d", l.Line, l.Column)
}
