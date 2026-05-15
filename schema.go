package kdl

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"regexp"
	"slices"
	"strings"
)

// A Schema is a parsed KDL schema document.
type Schema struct {
	Info              *SchemaInfo
	Nodes             []*SchemaNodeDef
	Tags              []*SchemaTagDef
	Definitions       *SchemaDefinitions
	NodeNames         *SchemaValidations
	OtherNodesAllowed bool
	TagNames          *SchemaValidations
	OtherTagsAllowed  bool
	ids               map[string]any // ID registry for ref resolution
}

// A SchemaInfo holds metadata about the schema itself.
type SchemaInfo struct {
	Title       string
	Description string
	Authors     []string
	Version     string
}

// A SchemaNodeDef describes the allowed structure of a KDL node. If Name is
// empty, the definition applies to all nodes.
type SchemaNodeDef struct {
	Name              string
	Description       string
	Id                string
	Ref               string
	Location          Location // start of the `node` token in the schema source
	NameEnd           Location // end (exclusive) of the `node` token in the schema source
	Min               *int
	Max               *int
	PropNames         *SchemaValidations
	OtherPropsAllowed bool
	Tag               *SchemaValidations
	Props             []*SchemaPropDef
	Values            []*SchemaValueDef
	rawChildren       []*SchemaChildrenDef // before merging/ref-resolution
	Children          *SchemaChildrenDef   // after resolving and merging
}

// A SchemaTagDef describes the allowed structure of a KDL type tag.
type SchemaTagDef struct {
	Name              string
	Description       string
	Id                string
	Ref               string
	Nodes             []*SchemaNodeDef
	NodeNames         *SchemaValidations
	OtherNodesAllowed bool
}

// A  SchemaPropDef describes a single property of a KDL node.
// If Key is empty, the definition applies to all properties.
type SchemaPropDef struct {
	Key         string
	Description string
	Id          string
	Ref         string
	Required    bool
	Location    Location // start of the `prop` token
	NameEnd     Location // end (exclusive) of the `prop` token
	Validations SchemaValidations
}

// A SchemaValueDef describes the positional values of a KDL node.
type SchemaValueDef struct {
	Description string
	Id          string
	Ref         string
	Min         *int
	Max         *int
	Location    Location // start of the `value` token
	NameEnd     Location // end (exclusive) of the `value` token
	Validations SchemaValidations
}

// A SchemaChildrenDef describes the allowed children of a KDL node.
type SchemaChildrenDef struct {
	Description       string
	Id                string
	Ref               string
	Nodes             []*SchemaNodeDef
	NodeNames         *SchemaValidations
	OtherNodesAllowed bool
}

// A SchemaDefinitions holds reusable node, tag, prop, value, and children definitions.
type SchemaDefinitions struct {
	Nodes    []*SchemaNodeDef
	Tags     []*SchemaTagDef
	Props    []*SchemaPropDef
	Values   []*SchemaValueDef
	Children []*SchemaChildrenDef
}

// A SchemaValidations holds the set of validation rules for a value or property.
type SchemaValidations struct {
	Types     []string // "string", "number", "boolean", "null"
	Enum      []Value
	Patterns  []string
	MinLength *int
	MaxLength *int
	Formats   []string
	Modulo    []Value
	Gt        *Value
	Gte       *Value
	Lt        *Value
	Lte       *Value
}

// ParseSchema parses a KDL schema document from r. The schema document must
// follow the structure defined in the KDL schema specification (version 1.0.0),
// with a top-level `document` node containing the schema definitions. Returns
// an error if parsing fails or if the document does not conform to the expected
// schema structure.
//
// Full KDL query support is not yet implemented—only [id="node-id"] queries are
// currently supported as references in `ref` properties.
func ParseSchema(r io.Reader) (*Schema, error) {
	doc, err := Parse(r)
	if err != nil {
		return nil, fmt.Errorf("kdl schema: parse error: %w", err)
	}
	return buildSchema(doc)
}

// ParseSchemaFromFile reads and parses a KDL schema document from path.
func ParseSchemaFromFile(path string) (*Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("kdl schema: open %q: %w", path, err)
	}
	defer f.Close()
	return ParseSchema(f)
}

// ValidateDocument validates doc against schema and returns diagnostics.
// The returned diagnostics all have Source set to "kdl-schema".
func ValidateDocument(doc *Document, schema *Schema) []Diagnostic {
	return validateChildren(doc.Nodes, nil, nil, schema.Nodes, schema.OtherNodesAllowed, schema)
}

// addRelated appends a DiagnosticRelated to d if start has a real location.
func addRelated(d *Diagnostic, start, end Location, msg string) {
	if start.Line == 0 {
		return
	}
	d.Related = append(d.Related, DiagnosticRelated{Start: start, End: end, Message: msg})
}

// withRelated returns diags with the given related info appended to each.
func withRelated(diags []Diagnostic, start, end Location, msg string) []Diagnostic {
	if start.Line == 0 {
		return diags
	}
	for i := range diags {
		diags[i].Related = append(diags[i].Related, DiagnosticRelated{Start: start, End: end, Message: msg})
	}
	return diags
}

var idRefRegex = regexp.MustCompile(`\[id="([^"]+)"\]`)

func extractRefId(ref string) string {
	m := idRefRegex.FindStringSubmatch(ref)
	if m == nil {
		return ""
	}
	return m[1]
}

func buildSchema(doc *Document) (*Schema, error) {
	docNode := doc.GetNode("document")
	if docNode == nil {
		return nil, fmt.Errorf("kdl schema: missing top-level 'document' node")
	}

	schema := &Schema{ids: make(map[string]any)}

	children := docNode.Children()
	if children == nil {
		return schema, nil
	}

	for _, child := range children.Nodes {
		switch child.Name() {
		case "info":
			info, err := parseSchemaInfo(child)
			if err != nil {
				return nil, err
			}
			schema.Info = info
		case "node":
			nd, err := parseSchemaNodeDef(child)
			if err != nil {
				return nil, err
			}
			schema.Nodes = append(schema.Nodes, nd)
		case "tag":
			td, err := parseSchemaTagDef(child)
			if err != nil {
				return nil, err
			}
			schema.Tags = append(schema.Tags, td)
		case "definitions":
			defs, err := parseSchemaDefinitions(child)
			if err != nil {
				return nil, err
			}
			schema.Definitions = defs
		case "node-names":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			schema.NodeNames = &v
		case "other-nodes-allowed":
			args := child.Arguments()
			if len(args) > 0 && args[0].Kind() == Bool {
				schema.OtherNodesAllowed = args[0].Bool()
			}
		case "tag-names":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			schema.TagNames = &v
		case "other-tags-allowed":
			args := child.Arguments()
			if len(args) > 0 && args[0].Kind() == Bool {
				schema.OtherTagsAllowed = args[0].Bool()
			}
		}
	}

	schema.buildIdRegistry()
	if err := schema.resolveRefs(); err != nil {
		return nil, err
	}
	schema.mergeChildren()

	return schema, nil
}

func parseSchemaInfo(n *Node) (*SchemaInfo, error) {
	info := &SchemaInfo{}
	children := n.Children()
	if children == nil {
		return info, nil
	}
	for _, child := range children.Nodes {
		args := child.Arguments()
		switch child.Name() {
		case "title":
			if len(args) > 0 && args[0].Kind() == String {
				info.Title = args[0].String()
			}
		case "description":
			if len(args) > 0 && args[0].Kind() == String {
				info.Description = args[0].String()
			}
		case "author", "contributor":
			if len(args) > 0 && args[0].Kind() == String {
				info.Authors = append(info.Authors, args[0].String())
			}
		case "version":
			if len(args) > 0 && args[0].Kind() == String {
				info.Version = args[0].String()
			}
		}
	}
	return info, nil
}

func parseSchemaNodeDef(n *Node) (*SchemaNodeDef, error) {
	def := &SchemaNodeDef{Location: n.Location(), NameEnd: n.NameEndLocation()}

	args := n.Arguments()
	if len(args) > 0 && args[0].Kind() == String {
		def.Name = args[0].String()
	}

	props := n.Properties()
	if v, ok := props["description"]; ok && v.Kind() == String {
		def.Description = v.String()
	}
	if v, ok := props["id"]; ok && v.Kind() == String {
		def.Id = v.String()
	}
	if v, ok := props["ref"]; ok && v.Kind() == String {
		def.Ref = v.String()
	}

	children := n.Children()
	if children == nil {
		return def, nil
	}

	for _, child := range children.Nodes {
		cargs := child.Arguments()
		switch child.Name() {
		case "min":
			if len(cargs) > 0 {
				if f, ok := toFloat64(cargs[0]); ok {
					v := int(f)
					def.Min = &v
				}
			}
		case "max":
			if len(cargs) > 0 {
				if f, ok := toFloat64(cargs[0]); ok {
					v := int(f)
					def.Max = &v
				}
			}
		case "prop-names":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			def.PropNames = &v
		case "other-props-allowed":
			if len(cargs) > 0 && cargs[0].Kind() == Bool {
				def.OtherPropsAllowed = cargs[0].Bool()
			}
		case "tag":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			def.Tag = &v
		case "prop":
			pd, err := parseSchemaPropDef(child)
			if err != nil {
				return nil, err
			}
			def.Props = append(def.Props, pd)
		case "value":
			vd, err := parseSchemaValueDef(child)
			if err != nil {
				return nil, err
			}
			def.Values = append(def.Values, vd)
		case "children":
			cd, err := parseSchemaChildrenDef(child)
			if err != nil {
				return nil, err
			}
			def.rawChildren = append(def.rawChildren, cd)
		}
	}

	return def, nil
}

func parseSchemaTagDef(n *Node) (*SchemaTagDef, error) {
	def := &SchemaTagDef{}

	args := n.Arguments()
	if len(args) > 0 && args[0].Kind() == String {
		def.Name = args[0].String()
	}

	props := n.Properties()
	if v, ok := props["description"]; ok && v.Kind() == String {
		def.Description = v.String()
	}
	if v, ok := props["id"]; ok && v.Kind() == String {
		def.Id = v.String()
	}
	if v, ok := props["ref"]; ok && v.Kind() == String {
		def.Ref = v.String()
	}

	children := n.Children()
	if children == nil {
		return def, nil
	}

	for _, child := range children.Nodes {
		cargs := child.Arguments()
		switch child.Name() {
		case "node":
			nd, err := parseSchemaNodeDef(child)
			if err != nil {
				return nil, err
			}
			def.Nodes = append(def.Nodes, nd)
		case "node-names":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			def.NodeNames = &v
		case "other-nodes-allowed":
			if len(cargs) > 0 && cargs[0].Kind() == Bool {
				def.OtherNodesAllowed = cargs[0].Bool()
			}
		}
	}

	return def, nil
}

func parseSchemaPropDef(n *Node) (*SchemaPropDef, error) {
	def := &SchemaPropDef{Location: n.Location(), NameEnd: n.NameEndLocation()}

	args := n.Arguments()
	if len(args) > 0 && args[0].Kind() == String {
		def.Key = args[0].String()
	}

	props := n.Properties()
	if v, ok := props["description"]; ok && v.Kind() == String {
		def.Description = v.String()
	}
	if v, ok := props["id"]; ok && v.Kind() == String {
		def.Id = v.String()
	}
	if v, ok := props["ref"]; ok && v.Kind() == String {
		def.Ref = v.String()
	}

	children := n.Children()
	if children == nil {
		return def, nil
	}

	// required node is prop-specific; rest are general validations
	var valNodes []*Node
	for _, child := range children.Nodes {
		if child.Name() == "required" {
			cargs := child.Arguments()
			if len(cargs) > 0 && cargs[0].Kind() == Bool {
				def.Required = cargs[0].Bool()
			}
		} else {
			valNodes = append(valNodes, child)
		}
	}

	v, err := parseSchemaValidationsFromNodes(valNodes)
	if err != nil {
		return nil, err
	}
	def.Validations = v

	return def, nil
}

func parseSchemaValueDef(n *Node) (*SchemaValueDef, error) {
	def := &SchemaValueDef{Location: n.Location(), NameEnd: n.NameEndLocation()}

	props := n.Properties()
	if v, ok := props["description"]; ok && v.Kind() == String {
		def.Description = v.String()
	}
	if v, ok := props["id"]; ok && v.Kind() == String {
		def.Id = v.String()
	}
	if v, ok := props["ref"]; ok && v.Kind() == String {
		def.Ref = v.String()
	}

	children := n.Children()
	if children == nil {
		return def, nil
	}

	// min/max are value-count constraints; rest are validations
	var valNodes []*Node
	for _, child := range children.Nodes {
		cargs := child.Arguments()
		switch child.Name() {
		case "min":
			if len(cargs) > 0 {
				if f, ok := toFloat64(cargs[0]); ok {
					v := int(f)
					def.Min = &v
				}
			}
		case "max":
			if len(cargs) > 0 {
				if f, ok := toFloat64(cargs[0]); ok {
					v := int(f)
					def.Max = &v
				}
			}
		default:
			valNodes = append(valNodes, child)
		}
	}

	v, err := parseSchemaValidationsFromNodes(valNodes)
	if err != nil {
		return nil, err
	}
	def.Validations = v

	return def, nil
}

func parseSchemaChildrenDef(n *Node) (*SchemaChildrenDef, error) {
	def := &SchemaChildrenDef{}

	props := n.Properties()
	if v, ok := props["description"]; ok && v.Kind() == String {
		def.Description = v.String()
	}
	if v, ok := props["id"]; ok && v.Kind() == String {
		def.Id = v.String()
	}
	if v, ok := props["ref"]; ok && v.Kind() == String {
		def.Ref = v.String()
	}

	children := n.Children()
	if children == nil {
		return def, nil
	}

	for _, child := range children.Nodes {
		cargs := child.Arguments()
		switch child.Name() {
		case "node":
			nd, err := parseSchemaNodeDef(child)
			if err != nil {
				return nil, err
			}
			def.Nodes = append(def.Nodes, nd)
		case "node-names":
			v, err := parseSchemaValidations(child.Children())
			if err != nil {
				return nil, err
			}
			def.NodeNames = &v
		case "other-nodes-allowed":
			if len(cargs) > 0 && cargs[0].Kind() == Bool {
				def.OtherNodesAllowed = cargs[0].Bool()
			}
		}
	}

	return def, nil
}

func parseSchemaDefinitions(n *Node) (*SchemaDefinitions, error) {
	defs := &SchemaDefinitions{}
	children := n.Children()
	if children == nil {
		return defs, nil
	}
	for _, child := range children.Nodes {
		switch child.Name() {
		case "node":
			nd, err := parseSchemaNodeDef(child)
			if err != nil {
				return nil, err
			}
			defs.Nodes = append(defs.Nodes, nd)
		case "tag":
			td, err := parseSchemaTagDef(child)
			if err != nil {
				return nil, err
			}
			defs.Tags = append(defs.Tags, td)
		case "prop":
			pd, err := parseSchemaPropDef(child)
			if err != nil {
				return nil, err
			}
			defs.Props = append(defs.Props, pd)
		case "value":
			vd, err := parseSchemaValueDef(child)
			if err != nil {
				return nil, err
			}
			defs.Values = append(defs.Values, vd)
		case "children":
			cd, err := parseSchemaChildrenDef(child)
			if err != nil {
				return nil, err
			}
			defs.Children = append(defs.Children, cd)
		}
	}
	return defs, nil
}

func parseSchemaValidations(doc *Document) (SchemaValidations, error) {
	if doc == nil {
		return SchemaValidations{}, nil
	}
	return parseSchemaValidationsFromNodes(doc.Nodes)
}

func parseSchemaValidationsFromNodes(nodes []*Node) (SchemaValidations, error) {
	var v SchemaValidations
	for _, child := range nodes {
		args := child.Arguments()
		switch child.Name() {
		case "type":
			for _, arg := range args {
				if arg.Kind() == String {
					v.Types = append(v.Types, arg.String())
				}
			}
		case "enum":
			v.Enum = append(v.Enum, args...)
		case "pattern":
			for _, arg := range args {
				if arg.Kind() == String {
					v.Patterns = append(v.Patterns, arg.String())
				}
			}
		case "min-length":
			if len(args) > 0 {
				if f, ok := toFloat64(args[0]); ok {
					n := int(f)
					v.MinLength = &n
				}
			}
		case "max-length":
			if len(args) > 0 {
				if f, ok := toFloat64(args[0]); ok {
					n := int(f)
					v.MaxLength = &n
				}
			}
		case "format":
			for _, arg := range args {
				if arg.Kind() == String {
					v.Formats = append(v.Formats, arg.String())
				}
			}
		case "%":
			v.Modulo = append(v.Modulo, args...)
		case ">":
			if len(args) > 0 {
				val := args[0]
				v.Gt = &val
			}
		case ">=":
			if len(args) > 0 {
				val := args[0]
				v.Gte = &val
			}
		case "<":
			if len(args) > 0 {
				val := args[0]
				v.Lt = &val
			}
		case "<=":
			if len(args) > 0 {
				val := args[0]
				v.Lte = &val
			}
		}
	}
	return v, nil
}

// ===== ID registry and ref resolution =====

func (s *Schema) buildIdRegistry() {
	var registerNodeDef func(nd *SchemaNodeDef)
	var registerChildrenDef func(cd *SchemaChildrenDef)

	registerPropDef := func(pd *SchemaPropDef) {
		if pd.Id != "" {
			s.ids[pd.Id] = pd
		}
	}

	registerValueDef := func(vd *SchemaValueDef) {
		if vd.Id != "" {
			s.ids[vd.Id] = vd
		}
	}

	registerChildrenDef = func(cd *SchemaChildrenDef) {
		if cd.Id != "" {
			s.ids[cd.Id] = cd
		}
		for _, nd := range cd.Nodes {
			registerNodeDef(nd)
		}
	}

	registerNodeDef = func(nd *SchemaNodeDef) {
		if nd.Id != "" {
			s.ids[nd.Id] = nd
		}
		for _, pd := range nd.Props {
			registerPropDef(pd)
		}
		for _, vd := range nd.Values {
			registerValueDef(vd)
		}
		for _, cd := range nd.rawChildren {
			registerChildrenDef(cd)
		}
	}

	registerTagDef := func(td *SchemaTagDef) {
		if td.Id != "" {
			s.ids[td.Id] = td
		}
		for _, nd := range td.Nodes {
			registerNodeDef(nd)
		}
	}

	for _, nd := range s.Nodes {
		registerNodeDef(nd)
	}
	for _, td := range s.Tags {
		registerTagDef(td)
	}
	if s.Definitions != nil {
		for _, nd := range s.Definitions.Nodes {
			registerNodeDef(nd)
		}
		for _, td := range s.Definitions.Tags {
			registerTagDef(td)
		}
		for _, pd := range s.Definitions.Props {
			registerPropDef(pd)
		}
		for _, vd := range s.Definitions.Values {
			registerValueDef(vd)
		}
		for _, cd := range s.Definitions.Children {
			registerChildrenDef(cd)
		}
	}
}

func (s *Schema) resolveRefs() error {
	visitingNodes := make(map[*SchemaNodeDef]bool)
	visitingChildren := make(map[*SchemaChildrenDef]bool)

	var resolveNodeDef func(nd *SchemaNodeDef) error
	var resolveChildrenDef func(cd *SchemaChildrenDef) error

	resolvePropDef := func(pd *SchemaPropDef) error {
		if pd.Ref == "" {
			return nil
		}
		id := extractRefId(pd.Ref)
		if id == "" {
			return nil
		}
		target, ok := s.ids[id]
		if !ok {
			// TODO: report missing refs
			return nil
		}
		ref, ok := target.(*SchemaPropDef)
		if !ok {
			// TODO: report invalid ref target type
			return nil
		}
		// ref provides defaults; ref wins on conflicts
		if len(ref.Validations.Types) > 0 {
			pd.Validations.Types = ref.Validations.Types
		}
		if len(ref.Validations.Enum) > 0 {
			pd.Validations.Enum = ref.Validations.Enum
		}
		if len(ref.Validations.Patterns) > 0 {
			pd.Validations.Patterns = ref.Validations.Patterns
		}
		if ref.Validations.MinLength != nil {
			pd.Validations.MinLength = ref.Validations.MinLength
		}
		if ref.Validations.MaxLength != nil {
			pd.Validations.MaxLength = ref.Validations.MaxLength
		}
		if len(ref.Validations.Formats) > 0 {
			pd.Validations.Formats = ref.Validations.Formats
		}
		if ref.Validations.Gt != nil {
			pd.Validations.Gt = ref.Validations.Gt
		}
		if ref.Validations.Gte != nil {
			pd.Validations.Gte = ref.Validations.Gte
		}
		if ref.Validations.Lt != nil {
			pd.Validations.Lt = ref.Validations.Lt
		}
		if ref.Validations.Lte != nil {
			pd.Validations.Lte = ref.Validations.Lte
		}
		if len(ref.Validations.Modulo) > 0 {
			pd.Validations.Modulo = ref.Validations.Modulo
		}
		if !pd.Required {
			pd.Required = ref.Required
		}
		return nil
	}

	resolveValueDef := func(vd *SchemaValueDef) error {
		if vd.Ref == "" {
			return nil
		}
		id := extractRefId(vd.Ref)
		if id == "" {
			return nil
		}
		target, ok := s.ids[id]
		if !ok {
			// TODO: report missing refs
			return nil
		}
		ref, ok := target.(*SchemaValueDef)
		if !ok {
			// TODO: report invalid ref target type
			return nil
		}
		if vd.Min == nil && ref.Min != nil {
			vd.Min = ref.Min
		}
		if vd.Max == nil && ref.Max != nil {
			vd.Max = ref.Max
		}
		if len(ref.Validations.Types) > 0 {
			vd.Validations.Types = ref.Validations.Types
		}
		if len(ref.Validations.Enum) > 0 {
			vd.Validations.Enum = ref.Validations.Enum
		}
		if len(ref.Validations.Patterns) > 0 {
			vd.Validations.Patterns = ref.Validations.Patterns
		}
		if ref.Validations.MinLength != nil {
			vd.Validations.MinLength = ref.Validations.MinLength
		}
		if ref.Validations.MaxLength != nil {
			vd.Validations.MaxLength = ref.Validations.MaxLength
		}
		if len(ref.Validations.Formats) > 0 {
			vd.Validations.Formats = ref.Validations.Formats
		}
		if ref.Validations.Gt != nil {
			vd.Validations.Gt = ref.Validations.Gt
		}
		if ref.Validations.Gte != nil {
			vd.Validations.Gte = ref.Validations.Gte
		}
		if ref.Validations.Lt != nil {
			vd.Validations.Lt = ref.Validations.Lt
		}
		if ref.Validations.Lte != nil {
			vd.Validations.Lte = ref.Validations.Lte
		}
		if len(ref.Validations.Modulo) > 0 {
			vd.Validations.Modulo = ref.Validations.Modulo
		}
		return nil
	}

	resolveChildrenDef = func(cd *SchemaChildrenDef) error {
		if visitingChildren[cd] {
			return nil
		}
		visitingChildren[cd] = true

		if cd.Ref != "" {
			id := extractRefId(cd.Ref)
			if id != "" {
				if target, ok := s.ids[id]; ok {
					if ref, ok := target.(*SchemaChildrenDef); ok {
						// prepend ref's nodes (ref wins)
						merged := make([]*SchemaNodeDef, 0, len(ref.Nodes)+len(cd.Nodes))
						merged = append(merged, ref.Nodes...)
						merged = append(merged, cd.Nodes...)
						cd.Nodes = merged
						if cd.NodeNames == nil && ref.NodeNames != nil {
							cd.NodeNames = ref.NodeNames
						}
						if !cd.OtherNodesAllowed {
							cd.OtherNodesAllowed = ref.OtherNodesAllowed
						}
					}
				}
			}
		}
		for _, nd := range cd.Nodes {
			if err := resolveNodeDef(nd); err != nil {
				return err
			}
		}
		return nil
	}

	resolveNodeDef = func(nd *SchemaNodeDef) error {
		if visitingNodes[nd] {
			return nil
		}
		visitingNodes[nd] = true

		if nd.Ref != "" {
			id := extractRefId(nd.Ref)
			if id != "" {
				if target, ok := s.ids[id]; ok {
					if ref, ok := target.(*SchemaNodeDef); ok {
						if nd.Name == "" && ref.Name != "" {
							nd.Name = ref.Name
						}
						if nd.Min == nil && ref.Min != nil {
							nd.Min = ref.Min
						}
						if nd.Max == nil && ref.Max != nil {
							nd.Max = ref.Max
						}
						if nd.PropNames == nil && ref.PropNames != nil {
							nd.PropNames = ref.PropNames
						}
						if !nd.OtherPropsAllowed {
							nd.OtherPropsAllowed = ref.OtherPropsAllowed
						}
						if nd.Tag == nil && ref.Tag != nil {
							nd.Tag = ref.Tag
						}
						if len(nd.Props) == 0 {
							nd.Props = ref.Props
						}
						if len(nd.Values) == 0 {
							nd.Values = ref.Values
						}
						if len(nd.rawChildren) == 0 {
							nd.rawChildren = ref.rawChildren
						}
					}
				}
			}
		}
		for _, pd := range nd.Props {
			if err := resolvePropDef(pd); err != nil {
				return err
			}
		}
		for _, vd := range nd.Values {
			if err := resolveValueDef(vd); err != nil {
				return err
			}
		}
		for _, cd := range nd.rawChildren {
			if err := resolveChildrenDef(cd); err != nil {
				return err
			}
		}
		return nil
	}

	for _, nd := range s.Nodes {
		if err := resolveNodeDef(nd); err != nil {
			return err
		}
	}
	for _, td := range s.Tags {
		for _, nd := range td.Nodes {
			if err := resolveNodeDef(nd); err != nil {
				return err
			}
		}
	}
	if s.Definitions != nil {
		for _, nd := range s.Definitions.Nodes {
			if err := resolveNodeDef(nd); err != nil {
				return err
			}
		}
		for _, pd := range s.Definitions.Props {
			if err := resolvePropDef(pd); err != nil {
				return err
			}
		}
		for _, vd := range s.Definitions.Values {
			if err := resolveValueDef(vd); err != nil {
				return err
			}
		}
		for _, cd := range s.Definitions.Children {
			if err := resolveChildrenDef(cd); err != nil {
				return err
			}
		}
	}

	return nil
}

// mergeChildren collapses all rawChildren blocks on each node def into a
// single Children field after ref resolution is complete.
func (s *Schema) mergeChildren() {
	visiting := make(map[*SchemaNodeDef]bool)
	var mergeNodeDef func(nd *SchemaNodeDef)

	mergeChildrenDef := func(blocks []*SchemaChildrenDef) *SchemaChildrenDef {
		if len(blocks) == 0 {
			return nil
		}
		merged := &SchemaChildrenDef{}
		for _, cd := range blocks {
			merged.Nodes = append(merged.Nodes, cd.Nodes...)
			if cd.OtherNodesAllowed {
				merged.OtherNodesAllowed = true
			}
			if merged.NodeNames == nil && cd.NodeNames != nil {
				merged.NodeNames = cd.NodeNames
			}
			if merged.Id == "" && cd.Id != "" {
				merged.Id = cd.Id
			}
		}
		return merged
	}

	mergeNodeDef = func(nd *SchemaNodeDef) {
		if visiting[nd] {
			return
		}
		visiting[nd] = true
		nd.Children = mergeChildrenDef(nd.rawChildren)
		if nd.Children != nil {
			for _, child := range nd.Children.Nodes {
				mergeNodeDef(child)
			}
		}
	}

	for _, nd := range s.Nodes {
		mergeNodeDef(nd)
	}
	for _, td := range s.Tags {
		for _, nd := range td.Nodes {
			mergeNodeDef(nd)
		}
	}
	if s.Definitions != nil {
		for _, nd := range s.Definitions.Nodes {
			mergeNodeDef(nd)
		}
	}
}

func validateChildren(nodes []*Node, parent *Node, parentDef *SchemaNodeDef, defs []*SchemaNodeDef, otherAllowed bool, schema *Schema) []Diagnostic {
	var diags []Diagnostic

	// count pass: check min/max for each def
	for _, def := range defs {
		if def.Name == "" {
			// wildcard def applies to all nodes
			count := len(nodes)
			if def.Min != nil && count < *def.Min {
				plural := ""
				if *def.Min != 1 {
					plural = "ren"
				}
				diag := Diagnostic{
					Severity: SeverityError,
					Message:  fmt.Sprintf("expected at least %d child%s, got %d", *def.Min, plural, count),
				}
				if parent != nil {
					diag.Start = parent.Location()
					diag.End = parent.NameEndLocation()
				}
				addRelated(&diag, def.Location, def.NameEnd, fmt.Sprintf("min=%d defined here", *def.Min))
				diags = append(diags, diag)
			}
			if def.Max != nil && count > *def.Max {
				plural := ""
				if *def.Max != 1 {
					plural = "ren"
				}
				diag := Diagnostic{
					Severity: SeverityError,
					Message:  fmt.Sprintf("expected at most %d child%s, got %d", *def.Max, plural, count),
				}
				if parent != nil {
					diag.Start = parent.Location()
					diag.End = parent.NameEndLocation()
				}
				addRelated(&diag, def.Location, def.NameEnd, fmt.Sprintf("max=%d defined here", *def.Max))
				diags = append(diags, diag)
			}
		} else {
			count := 0
			var lastMatchLoc, lastMatchEnd Location
			for _, n := range nodes {
				if n.Name() == def.Name {
					count++
					lastMatchLoc = n.Location()
					lastMatchEnd = n.NameEndLocation()
				}
			}
			if def.Min != nil && count < *def.Min {
				startLoc, endLoc := lastMatchLoc, lastMatchEnd
				if count == 0 && parent != nil {
					startLoc = parent.Location()
					endLoc = parent.NameEndLocation()
				}
				diag := Diagnostic{
					Start:    startLoc,
					End:      endLoc,
					Severity: SeverityError,
					Message:  fmt.Sprintf("node %q requires at least %d occurrence(s), got %d", def.Name, *def.Min, count),
				}
				addRelated(&diag, def.Location, def.NameEnd, fmt.Sprintf("node %q min=%d defined here", def.Name, *def.Min))
				diags = append(diags, diag)
			}
			if def.Max != nil && count > *def.Max {
				for _, n := range nodes {
					if n.Name() == def.Name {
						diag := Diagnostic{
							Start:    n.Location(),
							End:      n.NameEndLocation(),
							Severity: SeverityError,
							Message:  fmt.Sprintf("node %q allows at most %d occurrence(s)", def.Name, *def.Max),
						}
						addRelated(&diag, def.Location, def.NameEnd, fmt.Sprintf("node %q max=%d defined here", def.Name, *def.Max))
						diags = append(diags, diag)
					}
				}
			}
		}
	}

	// per-node validation pass
	for _, node := range nodes {
		def := findNodeDef(node.Name(), defs)
		if def == nil {
			if !otherAllowed {
				diag := Diagnostic{
					Start:    node.Location(),
					End:      node.NameEndLocation(),
					Severity: SeverityError,
					Message:  fmt.Sprintf("unexpected node %q", node.Name()),
				}
				if parentDef != nil {
					addRelated(&diag, parentDef.Location, parentDef.NameEnd, "allowed children defined here")
				}
				diags = append(diags, diag)
			}
			continue
		}
		diags = append(diags, validateNode(node, def, schema)...)
	}

	return diags
}

func findNodeDef(name string, defs []*SchemaNodeDef) *SchemaNodeDef {
	var wildcard *SchemaNodeDef
	for _, def := range defs {
		if def.Name == name {
			return def
		}
		if def.Name == "" {
			wildcard = def
		}
	}
	return wildcard
}

// FindSchemaNode returns the node def whose Name matches name, falling back to
// a wildcard def (one with an empty Name) if present. Returns nil if neither
// is found.
func FindSchemaNode(name string, defs []*SchemaNodeDef) *SchemaNodeDef {
	return findNodeDef(name, defs)
}

// FindSchemaProp returns the prop def whose Key matches key, falling back to
// a wildcard def (one with an empty Key) if present. Returns nil if neither
// is found.
func FindSchemaProp(key string, defs []*SchemaPropDef) *SchemaPropDef {
	return findPropDef(key, defs)
}

// ResolveNodePath walks the schema from root, descending one level per element
// of path. At each level the corresponding def is found via FindSchemaNode
// (exact name first, wildcard otherwise) and its merged Children.Nodes are
// used as the next level. Returns nil if any level cannot be resolved.
func (s *Schema) ResolveNodePath(path []string) *SchemaNodeDef {
	if s == nil || len(path) == 0 {
		return nil
	}
	defs := s.Nodes
	var def *SchemaNodeDef
	for _, name := range path {
		def = findNodeDef(name, defs)
		if def == nil {
			return nil
		}
		if def.Children == nil {
			defs = nil
		} else {
			defs = def.Children.Nodes
		}
	}
	return def
}

func validateNode(node *Node, def *SchemaNodeDef, schema *Schema) []Diagnostic {
	var diags []Diagnostic

	// validate props
	for _, propDef := range def.Props {
		if propDef.Key == "" {
			continue // wildcard checked below
		}
		if propDef.Required {
			if _, ok := node.Properties()[propDef.Key]; !ok {
				diag := Diagnostic{
					Start:    node.Location(),
					End:      node.NameEndLocation(),
					Severity: SeverityError,
					Message:  fmt.Sprintf("node %q missing required property %q", node.Name(), propDef.Key),
				}
				addRelated(&diag, propDef.Location, propDef.NameEnd, fmt.Sprintf("required property %q defined here", propDef.Key))
				diags = append(diags, diag)
			}
		}
	}

	for key, val := range node.Properties() {
		propDef := findPropDef(key, def.Props)
		if propDef == nil {
			if !def.OtherPropsAllowed {
				start, end, ok := node.PropertyKeyLocation(key)
				if !ok {
					start = node.Location()
					end = node.NameEndLocation()
				}
				diag := Diagnostic{
					Start:    start,
					End:      end,
					Severity: SeverityError,
					Message:  fmt.Sprintf("unexpected property %q on node %q", key, node.Name()),
				}
				addRelated(&diag, def.Location, def.NameEnd, fmt.Sprintf("node %q defined here", def.Name))
				diags = append(diags, diag)
			}
			continue
		}
		valDiags := validateValue(val, &propDef.Validations, val.Location(), val.EndLocation())
		valDiags = withRelated(valDiags, propDef.Location, propDef.NameEnd, fmt.Sprintf("property %q defined here", key))
		diags = append(diags, valDiags...)
	}

	// validate args
	args := node.Arguments()
	for _, valDef := range def.Values {
		if valDef.Min != nil && len(args) < *valDef.Min {
			diag := Diagnostic{
				Start:    node.Location(),
				End:      node.NameEndLocation(),
				Severity: SeverityError,
				Message:  fmt.Sprintf("node %q requires at least %d value(s), got %d", node.Name(), *valDef.Min, len(args)),
			}
			addRelated(&diag, valDef.Location, valDef.NameEnd, fmt.Sprintf("min=%d defined here", *valDef.Min))
			diags = append(diags, diag)
		}
		if valDef.Max != nil && len(args) > *valDef.Max {
			diag := Diagnostic{
				Start:    node.Location(),
				End:      node.NameEndLocation(),
				Severity: SeverityError,
				Message:  fmt.Sprintf("node %q allows at most %d value(s), got %d", node.Name(), *valDef.Max, len(args)),
			}
			addRelated(&diag, valDef.Location, valDef.NameEnd, fmt.Sprintf("max=%d defined here", *valDef.Max))
			diags = append(diags, diag)
		}
		for _, arg := range args {
			argDiags := validateValue(arg, &valDef.Validations, arg.Location(), arg.EndLocation())
			argDiags = withRelated(argDiags, valDef.Location, valDef.NameEnd, "value constraint defined here")
			diags = append(diags, argDiags...)
		}
	}

	// validate children recursively
	if def.Children != nil {
		childNodes := node.Children().Nodes
		diags = append(diags, validateChildren(childNodes, node, def, def.Children.Nodes, def.Children.OtherNodesAllowed, schema)...)
	}

	return diags
}

func findPropDef(key string, defs []*SchemaPropDef) *SchemaPropDef {
	var wildcard *SchemaPropDef
	for _, def := range defs {
		if def.Key == key {
			return def
		}
		if def.Key == "" {
			wildcard = def
		}
	}
	return wildcard
}

func validateValue(v Value, validations *SchemaValidations, loc, endLoc Location) []Diagnostic {
	if validations == nil {
		return nil
	}
	var diags []Diagnostic

	// type check
	if len(validations.Types) > 0 {
		kindName := valueKindToTypeName(v)
		valid := slices.Contains(validations.Types, kindName)
		if !valid {
			diags = append(diags, Diagnostic{
				Start:    loc,
				End:      endLoc,
				Severity: SeverityError,
				Message:  fmt.Sprintf("expected type %s, got %s", strings.Join(validations.Types, " or "), kindName),
			})
		}
	}

	// enum check
	if len(validations.Enum) > 0 {
		valid := false
		for _, e := range validations.Enum {
			if valueEqual(v, e) {
				valid = true
				break
			}
		}
		if !valid {
			diags = append(diags, Diagnostic{
				Start:    loc,
				End:      endLoc,
				Severity: SeverityError,
				Message:  fmt.Sprintf("value must be one of: %s", formatEnumValues(validations.Enum)),
			})
		}
	}

	// pattern check (strings only)
	if len(validations.Patterns) > 0 && v.Kind() == String {
		s := v.String()
		for _, pattern := range validations.Patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue // skip invalid pattern
			}
			if !re.MatchString(s) {
				diags = append(diags, Diagnostic{
					Start:    loc,
					End:      endLoc,
					Severity: SeverityError,
					Message:  fmt.Sprintf("value %q does not match pattern %q", s, pattern),
				})
			}
		}
	}

	// length checks (strings only)
	if v.Kind() == String {
		s := []rune(v.String())
		if validations.MinLength != nil && len(s) < *validations.MinLength {
			diags = append(diags, Diagnostic{
				Start:    loc,
				End:      endLoc,
				Severity: SeverityError,
				Message:  fmt.Sprintf("string length %d is less than minimum %d", len(s), *validations.MinLength),
			})
		}
		if validations.MaxLength != nil && len(s) > *validations.MaxLength {
			diags = append(diags, Diagnostic{
				Start:    loc,
				End:      endLoc,
				Severity: SeverityError,
				Message:  fmt.Sprintf("string length %d exceeds maximum %d", len(s), *validations.MaxLength),
			})
		}
	}

	// numeric bounds (numeric values only)
	if isNumeric(v) {
		f, ok := toFloat64(v)
		if ok {
			if validations.Gt != nil {
				if bound, ok2 := toFloat64(*validations.Gt); ok2 && f <= bound {
					diags = append(diags, Diagnostic{
						Start: loc, End: endLoc, Severity: SeverityError,
						Message: fmt.Sprintf("value must be > %s", formatValue(*validations.Gt)),
					})
				}
			}
			if validations.Gte != nil {
				if bound, ok2 := toFloat64(*validations.Gte); ok2 && f < bound {
					diags = append(diags, Diagnostic{
						Start: loc, End: endLoc, Severity: SeverityError,
						Message: fmt.Sprintf("value must be >= %s", formatValue(*validations.Gte)),
					})
				}
			}
			if validations.Lt != nil {
				if bound, ok2 := toFloat64(*validations.Lt); ok2 && f >= bound {
					diags = append(diags, Diagnostic{
						Start: loc, End: endLoc, Severity: SeverityError,
						Message: fmt.Sprintf("value must be < %s", formatValue(*validations.Lt)),
					})
				}
			}
			if validations.Lte != nil {
				if bound, ok2 := toFloat64(*validations.Lte); ok2 && f > bound {
					diags = append(diags, Diagnostic{
						Start: loc, End: endLoc, Severity: SeverityError,
						Message: fmt.Sprintf("value must be <= %s", formatValue(*validations.Lte)),
					})
				}
			}
			for _, mod := range validations.Modulo {
				if m, ok2 := toFloat64(mod); ok2 && m != 0 {
					if math.Abs(math.Mod(f, m)) > 1e-10 {
						diags = append(diags, Diagnostic{
							Start: loc, End: endLoc, Severity: SeverityError,
							Message: fmt.Sprintf("value must be a multiple of %s", formatValue(mod)),
						})
					}
				}
			}
		}
	}

	return diags
}

func valueKindToTypeName(v Value) string {
	switch v.Kind() {
	case String:
		return "string"
	case Int, BigInt, Float, BigFloat:
		return "number"
	case Bool:
		return "boolean"
	case Null:
		return "null"
	default:
		return "unknown"
	}
}

func valueEqual(a, b Value) bool {
	if a.Kind() != b.Kind() {
		// allow Int/Float cross-comparison for enum matching
		if isNumeric(a) && isNumeric(b) {
			fa, oka := toFloat64(a)
			fb, okb := toFloat64(b)
			return oka && okb && fa == fb
		}
		return false
	}
	switch a.Kind() {
	case String:
		return a.String() == b.String()
	case Bool:
		return a.Bool() == b.Bool()
	case Null:
		return true
	case Int:
		return a.Int() == b.Int()
	case Float:
		return a.Float() == b.Float()
	case BigInt:
		return a.BigInt().Cmp(b.BigInt()) == 0
	case BigFloat:
		return a.BigFloat().Cmp(b.BigFloat()) == 0
	default:
		return false
	}
}

func isNumeric(v Value) bool {
	switch v.Kind() {
	case Int, Float, BigInt, BigFloat:
		return true
	}
	return false
}

func toFloat64(v Value) (float64, bool) {
	switch v.Kind() {
	case Int:
		return float64(v.Int()), true
	case Float:
		return v.Float(), true
	case BigInt:
		f, _ := new(big.Float).SetInt(v.BigInt()).Float64()
		return f, true
	case BigFloat:
		f, _ := v.BigFloat().Float64()
		return f, true
	}
	return 0, false
}

func formatValue(v Value) string {
	switch v.Kind() {
	case String:
		return fmt.Sprintf("%q", v.String())
	case Bool:
		if v.Bool() {
			return "#true"
		}
		return "#false"
	case Null:
		return "#null"
	case Int:
		return fmt.Sprintf("%d", v.Int())
	case Float:
		return fmt.Sprintf("%g", v.Float())
	default:
		return v.String()
	}
}

func formatEnumValues(vals []Value) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatValue(v)
	}
	return strings.Join(parts, ", ")
}
