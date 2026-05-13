package kdl

// A ReservedTypeCategory groups reserved type annotations by what they apply
// to.
type ReservedTypeCategory int

const (
	// ReservedTypeStringFormat is a reserved annotation for string values.
	ReservedTypeStringFormat ReservedTypeCategory = iota
	// ReservedTypeNumberFormat is a reserved annotation for numeric values.
	ReservedTypeNumberFormat
)

func (c ReservedTypeCategory) String() string {
	switch c {
	case ReservedTypeStringFormat:
		return "string-format"
	case ReservedTypeNumberFormat:
		return "number-format"
	}
	return "unknown"
}

// A ReservedTypeAnnotation describes one reserved type annotation that may be
// used as a `format` value in a KDL schema or as a type annotation on a KDL
// value (e.g. `(u32)5`).
type ReservedTypeAnnotation struct {
	Name        string
	Description string
	Category    ReservedTypeCategory
}

// ReservedTypeAnnotations is the list of reserved type annotations defined by
// the KDL spec. This list may be changed to match future versions of the spec;
// consumers should not assume that any particular annotations are present, but
// may use FindReservedType() to check for specific annotations if desired.
var ReservedTypeAnnotations = []ReservedTypeAnnotation{
	// String formats
	{"date-time", "ISO8601 date/time format.", ReservedTypeStringFormat},
	{"time", "\"Time\" section of ISO8601.", ReservedTypeStringFormat},
	{"date", "\"Date\" section of ISO8601.", ReservedTypeStringFormat},
	{"duration", "ISO8601 duration format.", ReservedTypeStringFormat},
	{"decimal", "IEEE 754-2008 decimal string format.", ReservedTypeStringFormat},
	{"currency", "ISO 4217 currency code.", ReservedTypeStringFormat},
	{"country-2", "ISO 3166-1 alpha-2 country code.", ReservedTypeStringFormat},
	{"country-3", "ISO 3166-1 alpha-3 country code.", ReservedTypeStringFormat},
	{"country-subdivision", "ISO 3166-2 country subdivision code.", ReservedTypeStringFormat},
	{"email", "RFC5322 email address.", ReservedTypeStringFormat},
	{"idn-email", "RFC6531 internationalized email address.", ReservedTypeStringFormat},
	{"hostname", "RFC1123 internet hostname.", ReservedTypeStringFormat},
	{"idn-hostname", "RFC5890 internationalized internet hostname.", ReservedTypeStringFormat},
	{"ipv4", "RFC2673 dotted-quad IPv4 address.", ReservedTypeStringFormat},
	{"ipv6", "RFC2373 IPv6 address.", ReservedTypeStringFormat},
	{"url", "RFC3986 URI.", ReservedTypeStringFormat},
	{"url-reference", "RFC3986 URI Reference.", ReservedTypeStringFormat},
	{"irl", "RFC3987 Internationalized Resource Identifier.", ReservedTypeStringFormat},
	{"irl-reference", "RFC3987 Internationalized Resource Identifier Reference.", ReservedTypeStringFormat},
	{"url-template", "RFC6570 URI Template.", ReservedTypeStringFormat},
	{"uuid", "RFC4122 UUID.", ReservedTypeStringFormat},
	{"regex", "Regular expression. Specific patterns may be implementation-dependent.", ReservedTypeStringFormat},
	{"base64", "A Base64-encoded string, denoting arbitrary binary data.", ReservedTypeStringFormat},
	{"kdl-query", "A KDL Query string.", ReservedTypeStringFormat},

	// Number formats
	{"i8", "8-bit signed integer.", ReservedTypeNumberFormat},
	{"i16", "16-bit signed integer.", ReservedTypeNumberFormat},
	{"i32", "32-bit signed integer.", ReservedTypeNumberFormat},
	{"i64", "64-bit signed integer.", ReservedTypeNumberFormat},
	{"i128", "128-bit signed integer.", ReservedTypeNumberFormat},
	{"u8", "8-bit unsigned integer.", ReservedTypeNumberFormat},
	{"u16", "16-bit unsigned integer.", ReservedTypeNumberFormat},
	{"u32", "32-bit unsigned integer.", ReservedTypeNumberFormat},
	{"u64", "64-bit unsigned integer.", ReservedTypeNumberFormat},
	{"u128", "128-bit unsigned integer.", ReservedTypeNumberFormat},
	{"isize", "Platform-dependent signed integer.", ReservedTypeNumberFormat},
	{"usize", "Platform-dependent unsigned integer.", ReservedTypeNumberFormat},
	{"f32", "IEEE 754 single (32-bit) precision floating point number.", ReservedTypeNumberFormat},
	{"f64", "IEEE 754 double (64-bit) precision floating point number.", ReservedTypeNumberFormat},
	{"decimal64", "IEEE 754-2008 64-bit decimal floating point number.", ReservedTypeNumberFormat},
	{"decimal128", "IEEE 754-2008 128-bit decimal floating point number.", ReservedTypeNumberFormat},
}

// FindReservedType returns the reserved type annotation with the given name,
// or nil if no such reserved type exists.
func FindReservedType(name string) *ReservedTypeAnnotation {
	for i := range ReservedTypeAnnotations {
		if ReservedTypeAnnotations[i].Name == name {
			return &ReservedTypeAnnotations[i]
		}
	}
	return nil
}
