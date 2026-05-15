package kdl_test

import (
	"strings"
	"testing"

	kdl "github.com/calico32/kdl-go"
)

func TestExtractVersionDirective(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		want      kdl.Version
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "v1 before node",
			content:   "/- kdl-version 1\nnode1\n",
			want:      kdl.Version1,
			wantFound: true,
		},
		{
			name:      "v2 before node",
			content:   "/- kdl-version 2\nnode1\n",
			want:      kdl.Version2,
			wantFound: true,
		},
		{
			name:      "after blank lines and comments",
			content:   "// a comment\n\n/- kdl-version 2\nnode1\n",
			want:      kdl.Version2,
			wantFound: true,
		},
		{
			name:      "directive as trailing comment of document",
			content:   "node1\n/- kdl-version 1\n",
			want:      kdl.Version1,
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
			name:      "different slashdash node",
			content:   "/- kdl-schema \"./s.kdl\"\nnode\n",
			wantFound: false,
		},
		{
			name:      "missing arg",
			content:   "/- kdl-version\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "non-integer arg",
			content:   "/- kdl-version \"2\"\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "out-of-range version",
			content:   "/- kdl-version 3\nnode\n",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "too many args",
			content:   "/- kdl-version 1 2\nnode\n",
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
			dir, found := kdl.ExtractVersionDirective(doc)
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
				if dir.Version != kdl.VersionAuto {
					t.Errorf("Version = %v, want VersionAuto for malformed directive", dir.Version)
				}
				return
			}
			if dir.Err != "" {
				t.Fatalf("Err = %q, want empty", dir.Err)
			}
			if dir.Version != tc.want {
				t.Errorf("Version = %v, want %v", dir.Version, tc.want)
			}
			if dir.Start.Line == 0 {
				t.Errorf("Start.Line = 0, want directive location set")
			}
		})
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
