package kdl

// VersionDirective is the result of scanning a document for a
// `/- kdl-version <n>` marker directive via [ExtractVersionDirective]. The
// marker is the slashdashed node the KDL spec recommends at the top of a
// document to declare which spec version it targets.
type VersionDirective struct {
	// Version is the declared spec version ([Version1] or [Version2]) from a
	// well-formed directive. It is [VersionAuto] (the zero value) when Err is
	// set.
	Version Version
	// Start and End bound the whole `/- kdl-version ...` slashdash directive in
	// the source. They are valid whenever a directive was found, including a
	// malformed one.
	Start, End Location
	// Err describes why a found directive is malformed, or "" if it is valid.
	Err string
}

// ExtractVersionDirective scans the slashdashed nodes attached to a parsed
// document for a `/- kdl-version <n>` marker directive. The directive may
// appear as a leading comment of any top-level node or as a trailing comment of
// the document.
//
// The second return value is true if a slashdash node named "kdl-version" was
// found, whether or not it is well-formed. A well-formed directive has exactly
// one integer argument equal to 1 or 2 and no properties or children; a found
// directive that violates this has VersionDirective.Err set (and Version left
// at [VersionAuto]).
func ExtractVersionDirective(doc *Document) (VersionDirective, bool) {
	if doc == nil {
		return VersionDirective{}, false
	}
	scan := func(comments []Comment) (VersionDirective, bool) {
		for _, c := range comments {
			if c.Kind() != CommentSlashdash {
				continue
			}
			n := c.SlashedNode()
			if n == nil || n.Name() != "kdl-version" {
				continue
			}
			// c.Start() covers only the `/-` token; extend the range to the
			// end of the slashed node so callers see the whole directive
			end := n.EndLocation()
			if end.Line == 0 {
				end = c.End()
			}
			d := VersionDirective{Start: c.Start(), End: end}
			args := n.Arguments()
			switch {
			case len(args) == 0:
				d.Err = "kdl-version directive requires a single integer argument (1 or 2)"
			case len(args) > 1:
				d.Err = "kdl-version directive takes exactly one argument"
			case args[0].Kind() != Int:
				d.Err = "kdl-version directive argument must be an integer"
			case len(n.Properties()) > 0:
				d.Err = "kdl-version directive does not take properties"
			case n.Children() != nil && len(n.Children().Nodes) > 0:
				d.Err = "kdl-version directive does not take a children block"
			default:
				switch args[0].Int() {
				case 1:
					d.Version = Version1
				case 2:
					d.Version = Version2
				default:
					d.Err = "kdl-version directive must declare version 1 or 2"
				}
			}
			return d, true
		}
		return VersionDirective{}, false
	}
	for _, node := range doc.Nodes {
		if d, found := scan(node.LeadingComments()); found {
			return d, true
		}
	}
	if d, found := scan(doc.TrailingComments); found {
		return d, true
	}
	return VersionDirective{}, false
}

// SchemaDirective is the result of scanning a document for a
// `/- kdl-schema "<location>"` directive via [ExtractSchemaDirective].
type SchemaDirective struct {
	// Location is the schema location from a well-formed directive. Empty when
	// Err is set.
	Location string
	// Start and End bound the whole `/- kdl-schema ...` slashdash directive in
	// the source. They are valid whenever a directive was found, including a
	// malformed one.
	Start, End Location
	// Err describes why a found directive is malformed, or "" if it is valid.
	Err string
}

// ExtractSchemaDirective scans the slashdashed nodes attached to a parsed
// document for a `/- kdl-schema "<location>"` directive. The directive may
// appear as a leading comment of any top-level node or as a trailing comment of
// the document.
//
// Location may be a file path or a URI; it is up to callers to interpret it and
// load the referenced schema as needed.
//
// The second return value is true if a slashdash node named "kdl-schema" was
// found, whether or not it is well-formed. A well-formed directive has exactly
// one string argument and no properties or children; a found directive that
// violates this has SchemaDirective.Err set (and Path empty).
func ExtractSchemaDirective(doc *Document) (SchemaDirective, bool) {
	if doc == nil {
		return SchemaDirective{}, false
	}
	scan := func(comments []Comment) (SchemaDirective, bool) {
		for _, c := range comments {
			if c.Kind() != CommentSlashdash {
				continue
			}
			n := c.SlashedNode()
			if n == nil || n.Name() != "kdl-schema" {
				continue
			}
			// c.Start() covers only the `/-` token; extend the range to the
			// end of the slashed node so callers see the whole directive
			end := n.EndLocation()
			if end.Line == 0 {
				end = c.End()
			}
			d := SchemaDirective{Start: c.Start(), End: end}
			args := n.Arguments()
			switch {
			case len(args) == 0:
				d.Err = "kdl-schema directive requires a single string argument (the schema location)"
			case len(args) > 1:
				d.Err = "kdl-schema directive takes exactly one argument"
			case args[0].Kind() != String:
				d.Err = "kdl-schema directive argument must be a string"
			case len(n.Properties()) > 0:
				d.Err = "kdl-schema directive does not take properties"
			case n.Children() != nil && len(n.Children().Nodes) > 0:
				d.Err = "kdl-schema directive does not take a children block"
			default:
				d.Location = args[0].String()
			}
			return d, true
		}
		return SchemaDirective{}, false
	}
	for _, node := range doc.Nodes {
		if d, found := scan(node.LeadingComments()); found {
			return d, true
		}
	}
	if d, found := scan(doc.TrailingComments); found {
		return d, true
	}
	return SchemaDirective{}, false
}
