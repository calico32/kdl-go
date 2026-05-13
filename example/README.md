# kdl-go Examples

Examples of common use cases for `kdl-go`.

## [decode_encode](./decode_encode/decode_encode.go)

Decode a KDL config file into Go structs using struct tags, then encode it back to KDL.

## [flexible_decoder](./flexible_decoder/flexible_decoder.go)

Shows that the decoder accepts the same struct fields from node properties, children, or a mix of both — all produce identical results.

## [ast_builder](./ast_builder/ast_builder.go)

Build a KDL document from scratch using the programmatic AST API, then emit it.

## [walker](./walker/walker.go)

Traverse a KDL document tree in depth-first order, filtering and collecting nodes.

## [custom_types](./custom_types/custom_types.go)

Implement the `Marshaler` and `Unmarshaler` interfaces on a struct for full control over the KDL representation. Also shows `ValueMarshaler`/`ValueUnmarshaler` for custom scalar types.

## [diagnostics](./diagnostics/diagnostics.go)

Parse a malformed KDL document and inspect the error/warning diagnostics without aborting. The parser recovers and returns whatever nodes it could parse.

## [formatter](./formatter/formatter.go)

Format messy KDL with pretty-printer options, and emit with custom integer/string/version settings.
