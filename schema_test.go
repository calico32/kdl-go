package kdl_test

import (
	"os"
	"strings"
	"testing"

	kdl "github.com/calico32/kdl-go"
)

// parseTestSchema is a helper that parses a schema string and fatals on error.
func parseTestSchema(t *testing.T, src string) *kdl.Schema {
	t.Helper()
	s, err := kdl.ParseSchema(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	return s
}

// parseTestDoc is a helper that parses a KDL document string and fatals on error.
func parseTestDoc(t *testing.T, src string) *kdl.Document {
	t.Helper()
	doc, err := kdl.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

// hasError reports whether any diagnostic has SeverityError and its message
// contains the given substring. Pass "" to check for any error.
func hasError(diags []kdl.Diagnostic, substr string) bool {
	for _, d := range diags {
		if d.Severity == kdl.SeverityError {
			if substr == "" || strings.Contains(d.Message, substr) {
				return true
			}
		}
	}
	return false
}

func noErrors(t *testing.T, diags []kdl.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == kdl.SeverityError {
			t.Errorf("unexpected error diagnostic: %s", d.Message)
		}
	}
}

func TestExtractSchemaDirective(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		want      string
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "quoted path before node",
			content:   `/- kdl-schema "./my-schema.kdl"` + "\nnode1\n",
			want:      "./my-schema.kdl",
			wantFound: true,
		},
		{
			name:      "quoted path with spaces",
			content:   `/- kdl-schema "path/with spaces/schema.kdl"` + "\nnode1\n",
			want:      "path/with spaces/schema.kdl",
			wantFound: true,
		},
		{
			name:      "after blank lines and comments",
			content:   "// a comment\n\n" + `/- kdl-schema "./s.kdl"` + "\nnode1\n",
			want:      "./s.kdl",
			wantFound: true,
		},
		{
			name:      "directive as trailing comment of document",
			content:   "node1\n" + `/- kdl-schema "./s.kdl"` + "\n",
			want:      "./s.kdl",
			wantFound: true,
		},
		{
			name:      "no directive",
			content:   "node1\nnode2\n",
			wantFound: false,
		},
		{
			name:      "empty content",
			content:   "",
			wantFound: false,
		},
		{
			name:      "different comment content",
			content:   `/- kdl-version 2` + "\nnode\n",
			wantFound: false,
		},
		{
			name:      "missing arg",
			content:   "/- kdl-schema\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "wrong arg type",
			content:   "/- kdl-schema 1\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "too many args, props, and children",
			content:   `/- kdl-schema "a" "a" foo=1 { foo; }` + "\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "property only",
			content:   `/- kdl-schema "a" foo=1` + "\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "children only",
			content:   `/- kdl-schema "a" { foo; }` + "\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := kdl.Parse(strings.NewReader(tc.content))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			dir, found := kdl.ExtractSchemaDirective(doc)
			if found != tc.wantFound {
				t.Fatalf("found = %v, want %v", found, tc.wantFound)
			}
			if !found {
				return
			}
			if tc.wantErr {
				if dir.Err == "" {
					t.Errorf("Err = %q, want non-empty", dir.Err)
				}
				if dir.Location != "" {
					t.Errorf("Path = %q, want empty for malformed directive", dir.Location)
				}
				return
			}
			if dir.Err != "" {
				t.Fatalf("Err = %q, want empty", dir.Err)
			}
			if dir.Location != tc.want {
				t.Errorf("path = %q, want %q", dir.Location, tc.want)
			}
		})
	}
}

func TestParseSchema_MetaSchema(t *testing.T) {
	// parse the meta-schema (schema.kdl describes the schema format itself)
	s, err := kdl.ParseSchemaFromFile("schema/kdl-schema.kdl")
	if err != nil {
		t.Fatalf("ParseSchemaFromFile: %v", err)
	}
	if len(s.Nodes) == 0 {
		t.Error("expected at least one node def in meta-schema")
	}
	if s.Info == nil {
		t.Error("expected Info to be populated")
	}
}

func TestParseSchema_Minimal(t *testing.T) {
	s := parseTestSchema(t, `document {}`)
	if s == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestResolveNodePath(t *testing.T) {
	src := `document {
    node "server" {
        children {
            node "backend" {
                children {
                    node "db"
                }
            }
        }
    }
    node "wildcard" {
        children {
            node ""
        }
    }
}`
	s := parseTestSchema(t, src)

	if def := s.ResolveNodePath([]string{"server"}); def == nil || def.Name != "server" {
		t.Fatalf("server: got %v", def)
	}
	if def := s.ResolveNodePath([]string{"server", "backend", "db"}); def == nil || def.Name != "db" {
		t.Fatalf("server/backend/db: got %v", def)
	}
	if def := s.ResolveNodePath([]string{"server", "missing"}); def != nil {
		t.Fatalf("missing: got %v", def)
	}
	// wildcard match: an unknown child under "wildcard" should fall back to
	// the empty-name def
	if def := s.ResolveNodePath([]string{"wildcard", "anything"}); def == nil {
		t.Fatalf("wildcard fallback: got nil")
	}
	if def := s.ResolveNodePath(nil); def != nil {
		t.Fatalf("empty path: got %v", def)
	}
}

func TestParseSchema_MissingDocumentNode(t *testing.T) {
	_, err := kdl.ParseSchema(strings.NewReader(`something-else {}`))
	if err == nil {
		t.Fatal("expected error for missing 'document' node")
	}
}

func TestParseSchema_WithInfo(t *testing.T) {
	s := parseTestSchema(t, `
document {
    info {
        title "Test Schema"
        description "A test schema"
        version "1.0.0"
    }
}`)
	if s.Info == nil {
		t.Fatal("expected Info")
	}
	if s.Info.Title != "Test Schema" {
		t.Errorf("title = %q, want %q", s.Info.Title, "Test Schema")
	}
	if s.Info.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", s.Info.Version, "1.0.0")
	}
}

func TestValidateDocument_MetaSchema(t *testing.T) {
	s, err := kdl.ParseSchemaFromFile("schema/kdl-schema.kdl")
	if err != nil {
		t.Fatalf("ParseSchemaFromFile: %v", err)
	}

	// schema should pass validation against itself
	f, err := os.Open("schema/kdl-schema.kdl")
	if err != nil {
		t.Fatalf("failed to open schema.kdl: %v", err)
	}
	defer f.Close()

	doc, err := kdl.Parse(f)
	if err != nil {
		t.Fatalf("failed to parse schema.kdl: %v", err)
	}

	diags := kdl.ValidateDocument(doc, s)
	noErrors(t, diags)
}

func TestValidateDocument_Valid(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "server" {
        min 1
        max 1
        prop "host" {
            required #true
            type string
        }
        prop "port" {
            type number
        }
        value {
            min 0
            max 0
        }
    }
}`)

	doc := parseTestDoc(t, `server host="localhost" port=8080`)
	diags := kdl.ValidateDocument(doc, s)
	noErrors(t, diags)
}

func TestValidateDocument_UnexpectedNode(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "allowed" {}
}`)

	doc := parseTestDoc(t, `allowed; unexpected`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, "unexpected node") {
		t.Error("expected 'unexpected node' error for unknown node name")
	}
}

func TestValidateDocument_OtherNodesAllowed(t *testing.T) {
	s := parseTestSchema(t, `
document {
    other-nodes-allowed #true
    node "known" {}
}`)

	doc := parseTestDoc(t, `known; unknown1; unknown2`)
	diags := kdl.ValidateDocument(doc, s)
	noErrors(t, diags)
}

func TestValidateDocument_NodeMinCount(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "required-node" {
        min 1
    }
}`)

	doc := parseTestDoc(t, `other-node`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, "required-node") {
		t.Error("expected error about missing required node")
	}
}

func TestValidateDocument_NodeMaxCount(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "singleton" {
        max 1
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `singleton; singleton`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, "singleton") {
		t.Error("expected error about max occurrences exceeded")
	}
}

func TestValidateDocument_RequiredProp(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "name" {
            required #true
            type string
        }
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `node`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, `"name"`) {
		t.Error("expected error about missing required property 'name'")
	}
}

func TestValidateDocument_RequiredPropPresent(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "name" {
            required #true
            type string
        }
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `node name="Alice"`)
	diags := kdl.ValidateDocument(doc, s)
	noErrors(t, diags)
}

func TestValidateDocument_UnexpectedProp(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "known" {}
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `node known="x" unknown="y"`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, `"unknown"`) {
		t.Error("expected error about unexpected property 'unknown'")
	}
}

func TestValidateDocument_OtherPropsAllowed(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        other-props-allowed #true
        prop "known" {}
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `node known="x" anything="y" foo=42`)
	diags := kdl.ValidateDocument(doc, s)
	noErrors(t, diags)
}

func TestValidateDocument_PropTypeString(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        other-props-allowed #true
        prop "name" {
            type string
        }
    }
    other-nodes-allowed #true
}`)

	doc := parseTestDoc(t, `node name=42`)
	diags := kdl.ValidateDocument(doc, s)
	if !hasError(diags, "type") {
		t.Error("expected type mismatch error")
	}
}

func TestValidateDocument_PropTypeNumber(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "count" {
            type number
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node count=42`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node count="not-a-number"`)
	if !hasError(kdl.ValidateDocument(docBad, s), "type") {
		t.Error("expected type mismatch error for string where number expected")
	}
}

func TestValidateDocument_PropTypeBool(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "flag" {
            type boolean
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node flag=#true`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node flag="yes"`)
	if !hasError(kdl.ValidateDocument(docBad, s), "type") {
		t.Error("expected type mismatch for string where boolean expected")
	}
}

func TestValidateDocument_PropEnum(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "color" {
            type string
            enum "red" "green" "blue"
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node color="red"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node color="yellow"`)
	if !hasError(kdl.ValidateDocument(docBad, s), "one of") {
		t.Error("expected enum validation error")
	}
}

func TestValidateDocument_ValueEnum(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "level" {
        value {
            min 1
            max 1
            type string
            enum "low" "medium" "high"
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `level "high"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `level "extreme"`)
	if !hasError(kdl.ValidateDocument(docBad, s), "one of") {
		t.Error("expected enum error for value")
	}
}

func TestValidateDocument_Pattern(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "email" {
            type string
            pattern "^[^@]+@[^@]+$"
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node email="user@example.com"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node email="not-an-email"`)
	if !hasError(kdl.ValidateDocument(docBad, s), "pattern") {
		t.Error("expected pattern validation error")
	}
}

func TestValidateDocument_MinMaxLength(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "code" {
            type string
            min-length 3
            max-length 5
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node code="abc"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docTooShort := parseTestDoc(t, `node code="ab"`)
	if !hasError(kdl.ValidateDocument(docTooShort, s), "minimum") {
		t.Error("expected min-length error")
	}

	docTooLong := parseTestDoc(t, `node code="toolong"`)
	if !hasError(kdl.ValidateDocument(docTooLong, s), "maximum") {
		t.Error("expected max-length error")
	}
}

func TestValidateDocument_NumericBoundsGT(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "port" {
            type number
            ">" 0
            "<=" 65535
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node port=8080`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docZero := parseTestDoc(t, `node port=0`)
	if !hasError(kdl.ValidateDocument(docZero, s), "> 0") {
		t.Error("expected > 0 violation")
	}

	docOver := parseTestDoc(t, `node port=99999`)
	if !hasError(kdl.ValidateDocument(docOver, s), "<= 65535") {
		t.Error("expected <= 65535 violation")
	}
}

func TestValidateDocument_NumericBoundsGTE(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "score" {
            type number
            ">=" 0
            "<" 100
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node score=0`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docNeg := parseTestDoc(t, `node score=-1`)
	if !hasError(kdl.ValidateDocument(docNeg, s), ">= 0") {
		t.Error("expected >= 0 violation")
	}

	docAt100 := parseTestDoc(t, `node score=100`)
	if !hasError(kdl.ValidateDocument(docAt100, s), "< 100") {
		t.Error("expected < 100 violation")
	}
}

func TestValidateDocument_Modulo(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop "step" {
            type number
            % 5
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node step=15`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node step=7`)
	if !hasError(kdl.ValidateDocument(docBad, s), "multiple") {
		t.Error("expected modulo validation error")
	}
}

func TestValidateDocument_ValueMinMax(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "triple" {
        value {
            min 3
            max 3
            type number
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `triple 1 2 3`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docTooFew := parseTestDoc(t, `triple 1 2`)
	if !hasError(kdl.ValidateDocument(docTooFew, s), "at least 3") {
		t.Error("expected value min error")
	}

	docTooMany := parseTestDoc(t, `triple 1 2 3 4`)
	if !hasError(kdl.ValidateDocument(docTooMany, s), "at most 3") {
		t.Error("expected value max error")
	}
}

func TestValidateDocument_NestedChildren(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "server" {
        children {
            node "route" {
                min 1
                prop "path" {
                    required #true
                    type string
                }
                children {
                    node "handler" {
                        max 1
                        value {
                            min 1
                            max 1
                            type string
                        }
                    }
                    other-nodes-allowed #false
                }
            }
            other-nodes-allowed #false
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `
server {
    route path="/api" {
        handler "myHandler"
    }
}`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docMissingPath := parseTestDoc(t, `
server {
    route {
        handler "myHandler"
    }
}`)
	if !hasError(kdl.ValidateDocument(docMissingPath, s), `"path"`) {
		t.Error("expected error for missing required 'path' prop")
	}

	docUnexpectedChild := parseTestDoc(t, `
server {
    route path="/api" {
        middleware "cors"
    }
}`)
	if !hasError(kdl.ValidateDocument(docUnexpectedChild, s), "unexpected node") {
		t.Error("expected error for unexpected child node 'middleware'")
	}
}

func TestValidateDocument_WildcardNodeDef(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node {
        prop "required-prop" {
            required #true
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `any-node required-prop="x"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `any-node`)
	if !hasError(kdl.ValidateDocument(docBad, s), `"required-prop"`) {
		t.Error("expected error from wildcard node def for missing prop")
	}
}

func TestValidateDocument_WildcardPropDef(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node" {
        prop {
            type string
        }
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node a="x" b="y"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node a=42`)
	if !hasError(kdl.ValidateDocument(docBad, s), "type") {
		t.Error("expected type error from wildcard prop def")
	}
}

func TestValidateDocument_RefResolution(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "node-a" {
        prop "shared" id="shared-prop" {
            type string
            min-length 1
        }
    }
    node "node-b" {
        prop "shared" ref="[id=\"shared-prop\"]"
    }
    other-nodes-allowed #true
}`)

	docOk := parseTestDoc(t, `node-a shared="hello"
node-b shared="world"`)
	noErrors(t, kdl.ValidateDocument(docOk, s))

	docBad := parseTestDoc(t, `node-b shared=42`)
	if !hasError(kdl.ValidateDocument(docBad, s), "type") {
		t.Error("expected type error via resolved ref")
	}
}

func TestValidateDocument_MultipleTypes(t *testing.T) {
	s := parseTestSchema(t, `
document {
    node "flexible" {
        value {
            type string
            type number
        }
    }
    other-nodes-allowed #true
}`)

	docStr := parseTestDoc(t, `flexible "hello"`)
	noErrors(t, kdl.ValidateDocument(docStr, s))

	docNum := parseTestDoc(t, `flexible 42`)
	noErrors(t, kdl.ValidateDocument(docNum, s))

	docBool := parseTestDoc(t, `flexible #true`)
	if !hasError(kdl.ValidateDocument(docBool, s), "type") {
		t.Error("expected type error for boolean when only string/number allowed")
	}
}
