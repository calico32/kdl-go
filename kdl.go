// Package kdl implements a parser and emitter for [KDL] documents. It supports
// decoding KDL documents into a structured representation, unmarshaling them
// into Go data structures, and marshaling/encoding them back into KDL.
//
// Both KDL version 1.0.0 and 2.0.0 are supported, and the parser passes the
// upstream test suite for each. Note that the parser primarily targets the v2
// spec and is somewhat more permissive when parsing v1 input.
//
// [Encode] and [Decode] are the primary entry points for encoding and decoding
// KDL documents to and from Go data structures. These functions encode and
// decode Go structs, slices, maps, and scalars using conventions similar to
// those used by encoding/json. Use struct tags to customize the mapping between
// KDL and Go types, or implement any of [DocumentMarshaler],
// [DocumentUnmarshaler], [Marshaler], [Unmarshaler], [ValueMarshaler], or
// [ValueUnmarshaler] for more control over the process.
//
// For lower-level access, [Parse] and [Emit] can be used to parse and emit KDL
// documents from and to an in-memory representation, defined by the [Document],
// [Node], and [Value] types. These types can also be used directly to build or
// manipulate KDL documents programmatically without marshaling or unmarshaling.
// Use [NewDocument], [NewNode], [NewKV], [NewValue], and related functions and
// methods to create KDL documents, nodes, key-value nodes, and values.
//
// Other features include pretty-printing via [Format], AST traversal via
// [Walk], and KDL Schema validation via [ParseSchema] and [ValidateDocument].
//
// [KDL]: https://kdl.dev/
package kdl

import "fmt"

type Version int

const (
	// VersionAuto emits at v2 and parses at v2, but falls back to v1 if
	// parsing fails.
	VersionAuto Version = 0
	// Version2 is KDL version 2.0.0, adhering to https://kdl.dev/spec.
	Version2 Version = 2
	// Version1 is KDL version 1.0.0, (somewhat loosely) adhering to
	// https://kdl.dev/spec-v1.
	Version1 Version = 1
)

func (v Version) String() string {
	switch v {
	case VersionAuto:
		return "auto"
	case Version2:
		return "v2"
	case Version1:
		return "v1"
	default:
		return fmt.Sprintf("unknown version %d", v)
	}
}
