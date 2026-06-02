package test

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"path/filepath"
	"time"
)

//go:embed kdl1/tests/test_cases/*
var kdl1Tests embed.FS

var Kdl1Tests = fixedV1TestSuite{kdl1Tests}

//go:embed kdl2/tests/test_cases/*
var Kdl2Tests embed.FS

// hack to patch the v1 test suite files with correct content for some known issues

type fixedV1TestSuite struct{ embed.FS }

var _ fs.ReadDirFS = fixedV1TestSuite{}
var _ fs.ReadFileFS = fixedV1TestSuite{}

func (a fixedV1TestSuite) ReadFile(name string) ([]byte, error) {
	f, err := a.Open(name)
	if err != nil {
		return nil, err
	}
	if f, ok := f.(file); ok {
		return f.Bytes(), nil
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (a fixedV1TestSuite) Open(name string) (fs.File, error) {
	var content string
	switch name {
	case "kdl1/tests/test_cases/expected_kdl/underscore_in_fraction.kdl":
		// missing
		content = `node 1.02`
	case "kdl1/tests/test_cases/input/unusual_bare_id_chars_in_quoted_id.kdl":
		// incorrect (/ not allowed in bare id)
		content = `"foo123~!@#$%^&*.:'|?+" "weeee"`
	case "kdl1/tests/test_cases/expected_kdl/unusual_bare_id_chars_in_quoted_id.kdl",
		"kdl1/tests/test_cases/expected_kdl/unusual_chars_in_bare_id.kdl",
		"kdl1/tests/test_cases/input/unusual_chars_in_bare_id.kdl":
		// incorrect (/ not allowed in bare id)
		content = `foo123~!@#$%^&*.:'|?+ "weeee"`
	}
	if content != "" {
		return file{filepath.Base(name), bytes.NewBufferString(content)}, nil
	}

	return a.FS.Open(name)
}

type file struct {
	name string
	*bytes.Buffer
}

var _ fs.File = file{}

func (f file) Stat() (fs.FileInfo, error) { return f, nil }
func (f file) Close() error               { return nil }

var _ fs.FileInfo = file{}

func (f file) Name() string       { return f.name }
func (f file) Size() int64        { return int64(f.Len()) }
func (f file) IsDir() bool        { return false }
func (f file) ModTime() time.Time { panic("internal/test.file: ModTime unimplemented") }
func (f file) Mode() fs.FileMode  { panic("internal/test.file: Mode unimplemented") }
func (f file) Sys() any           { return nil }
