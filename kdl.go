// Package kdl implements a parser and emitter for [KDL] documents. It supports
// decoding KDL documents into a structured representation, unmarshaling them
// into Go data structures, and marshaling/encoding them back into KDL.
//
// Only KDL version 2.0.0 is supported. The parser is compliant with the
// KDL 2.0.0 specification and passes the official KDL test suite.
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
// [KDL]: https://kdl.dev/
package kdl
