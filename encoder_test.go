package kdl_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/calico32/kdl-go"
)

type EncoderTimes struct {
	DefaultT  time.Time     `kdl:"default-t"`
	RFC3339T  time.Time     `kdl:"rfc-t,format:RFC3339"`
	UnixT     time.Time     `kdl:"unix-t,format:unix"`
	UnixMsT   time.Time     `kdl:"unix-ms-t,format:unixmilli"`
	DateOnlyT time.Time     `kdl:"date-t,format:DateOnly"`
	UnitsD    time.Duration `kdl:"units-d"`
	SecD      time.Duration `kdl:"sec-d,format:sec"`
	MilliD    time.Duration `kdl:"milli-d,format:milli"`
	NanoD     time.Duration `kdl:"nano-d,format:nano"`
}

type EncoderPerson struct {
	Name    string `kdl:"name"`
	Age     int    `kdl:"age,omitzero"`
	Married bool   `kdl:"married,omitzero"`
}

type EncoderBook struct {
	Title  string        `kdl:"title"`
	Author EncoderPerson `kdl:"author,omitzero"`
}

type EncoderPeople struct {
	People []EncoderPersonProps `kdl:"person,omitzero,multiple"`
}

type EncoderPersonProps struct {
	Name    string            `kdl:",arg"`
	Age     int               `kdl:"age,prop,omitzero"`
	Married bool              `kdl:"married,prop,omitzero"`
	Extra   map[string]string `kdl:",props,omitzero"`
	Args    []string          `kdl:",args,omitzero"`
}

type EncoderPeopleGroups struct {
	Groups []EncoderMapStruct `kdl:"group,multiple"`
}

type EncoderMapStruct struct {
	Name   string                          `kdl:",arg"`
	People map[string]EncoderPersonDetails `kdl:",children"`
}

type EncoderPersonDetails struct {
	Age     int  `kdl:"age,prop"`
	Married bool `kdl:"married,prop"`
}

type EncoderCustomNodeMarshaler struct {
	Foo string
	Bar int
}

var _ kdl.Marshaler = &EncoderCustomNodeMarshaler{}

func (e *EncoderCustomNodeMarshaler) MarshalKDL() (*kdl.Node, error) {
	node := kdl.NewNode("custom")
	node.AddArgument(kdl.NewString(e.Foo + "-custom"))
	node.AddProperty("bar", kdl.NewInt(e.Bar*10))
	return node, nil
}

type EncoderCustomValueMarshaler string

var _ kdl.ValueMarshaler = EncoderCustomValueMarshaler("")

func (e EncoderCustomValueMarshaler) MarshalKDLValue() (kdl.Value, error) {
	return kdl.NewString("value-" + string(e)), nil
}

type EncoderCustomDocumentMarshaler struct {
	Title string
	Count int
}

var _ kdl.DocumentMarshaler = &EncoderCustomDocumentMarshaler{}

func (e *EncoderCustomDocumentMarshaler) MarshalKDLDocument() (*kdl.Document, error) {
	doc := &kdl.Document{}
	doc.Nodes = append(doc.Nodes, kdl.NewKV("custom-title", e.Title))
	doc.Nodes = append(doc.Nodes, kdl.NewKV("custom-count", e.Count))
	return doc, nil
}

var encoderTests = []struct {
	name     string
	value    any
	expected string
}{
	{"basic",
		EncoderPerson{
			Name:    "Alice",
			Age:     30,
			Married: true,
		},
		`
			name Alice
			age 30
			married #true
		`,
	},
	{"omitzero",
		EncoderPerson{
			Name: "Bob",
		},
		`
			name Bob
		`,
	},
	{"nested struct",
		EncoderBook{
			Title: "Go Programming",
			Author: EncoderPerson{
				Name:    "Carol",
				Age:     28,
				Married: true,
			},
		},
		`
			title "Go Programming"
			author {
				name Carol
				age 28
				married #true
			}
		`,
	},
	{"flags",
		EncoderPeople{
			People: []EncoderPersonProps{
				{Name: "Dave", Age: 40, Extra: map[string]string{"job": "Engineer", "hobby": "golf", "foo": ""}},
				{Name: "Eve", Married: true, Args: []string{"extra1", "", "extra2"}},
			},
		},
		`
			person Dave age=40 job=Engineer hobby=golf
			person Eve married=#true extra1 extra2
		`,
	},
	{"map of structs",
		EncoderPeopleGroups{
			Groups: []EncoderMapStruct{
				{
					Name: "Group1",
					People: map[string]EncoderPersonDetails{
						"Frank": {Age: 35, Married: false},
						"Grace": {Age: 29, Married: true},
					},
				},
				{
					Name: "Group2",
					People: map[string]EncoderPersonDetails{
						"Heidi": {Age: 32, Married: true},
					},
				},
			},
		},
		`
			group Group1 {
				Frank age=35 married=#false
				Grace age=29 married=#true
			}
			group Group2 {
				Heidi age=32 married=#true
			}
		`,
	},
	{
		"custom marshalers",
		struct {
			Node  EncoderCustomNodeMarshaler     `kdl:"node"`
			Value EncoderCustomValueMarshaler    `kdl:"value"`
			Doc   EncoderCustomDocumentMarshaler `kdl:"doc"`
		}{
			Node: EncoderCustomNodeMarshaler{
				Foo: "test",
				Bar: 7,
			},
			Value: EncoderCustomValueMarshaler("qux"),
			Doc: EncoderCustomDocumentMarshaler{
				Title: "My Document",
				Count: 3,
			},
		},
		`
			custom test-custom bar=70
			value value-qux
			doc {
				custom-title "My Document"
				custom-count 3
			}
		`,
	},
	{
		"time encoding",
		EncoderTimes{
			DefaultT:  time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
			RFC3339T:  time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
			UnixT:     time.Unix(1704207845, 0).UTC(),
			UnixMsT:   time.Unix(1704207845, 500_000_000).UTC(),
			DateOnlyT: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			UnitsD:    2*time.Hour + 30*time.Minute,
			SecD:      90 * time.Second,
			MilliD:    1500 * time.Millisecond,
			NanoD:     250 * time.Nanosecond,
		},
		`
			default-t "2024-01-02T15:04:05Z"
			rfc-t "2024-01-02T15:04:05Z"
			unix-t 1704207845
			unix-ms-t 1704207845500
			date-t "2024-01-02"
			units-d "2h30m0s"
			sec-d 90
			milli-d 1500
			nano-d 250
		`,
	},
	{
		"document marshaler",
		&EncoderCustomDocumentMarshaler{
			Title: "Document Only",
			Count: 42,
		},
		`
			custom-title "Document Only"
			custom-count 42
		`,
	},
	{
		"omitzero with Located",
		struct {
			Name kdl.Located[string] `kdl:"name,omitzero"`
		}{
			Name: kdl.Located[string]{Value: "", Start: kdl.Location{Line: 1, Column: 1}, End: kdl.Location{Line: 1, Column: 2}},
		},
		``,
	},
}

func TestEncoder(t *testing.T) {
	for _, tt := range encoderTests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := kdl.Marshal(tt.value)
			if err != nil {
				t.Errorf("unexpected error during marshaling: %+v", err)
				return
			}
			actualStr := new(bytes.Buffer)
			err = kdl.Emit(doc, actualStr)
			if err != nil {
				t.Errorf("unexpected error during emitting: %+v", err)
				return
			}

			actual, err := roundtrip(actualStr.String())
			if err != nil {
				t.Errorf("unexpected error during actual roundtrip: %+v", err)
				return
			}

			expected, err := roundtrip(tt.expected)
			if err != nil {
				t.Errorf("unexpected error during expected roundtrip: %+v", err)
				return
			}

			if actual != expected {
				t.Errorf("marshaled document does not match expected\nExpected:\n%s\nGot:\n%s", expected, actual)
			}
		})
	}
}

func TestEncodeDecodeTimeRoundtrip(t *testing.T) {
	original := EncoderTimes{
		DefaultT:  time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		RFC3339T:  time.Date(2024, 1, 2, 15, 4, 5, 0, time.Local),
		UnixT:     time.Unix(1704207845, 0).UTC(),
		UnixMsT:   time.Unix(1704207845, 500_000_000).UTC(),
		DateOnlyT: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		UnitsD:    2*time.Hour + 30*time.Minute,
		SecD:      90 * time.Second,
		MilliD:    1500 * time.Millisecond,
		NanoD:     250 * time.Nanosecond,
	}

	s, err := kdl.EncodeToString(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var decoded EncoderTimes
	if err := kdl.Decode(strings.NewReader(s), &decoded); err != nil {
		t.Fatalf("decode: %v\nKDL was:\n%s", err, s)
	}

	if !decoded.DefaultT.Equal(original.DefaultT) {
		t.Errorf("DefaultT: got %v, want %v", decoded.DefaultT, original.DefaultT)
	}
	if !decoded.UnixT.Equal(original.UnixT) {
		t.Errorf("UnixT: got %v, want %v", decoded.UnixT, original.UnixT)
	}
	if !decoded.UnixMsT.Equal(original.UnixMsT) {
		t.Errorf("UnixMsT: got %v, want %v", decoded.UnixMsT, original.UnixMsT)
	}
	if decoded.UnitsD != original.UnitsD {
		t.Errorf("UnitsD: got %v, want %v", decoded.UnitsD, original.UnitsD)
	}
	if decoded.SecD != original.SecD {
		t.Errorf("SecD: got %v, want %v", decoded.SecD, original.SecD)
	}
	if decoded.MilliD != original.MilliD {
		t.Errorf("MilliD: got %v, want %v", decoded.MilliD, original.MilliD)
	}
	if decoded.NanoD != original.NanoD {
		t.Errorf("NanoD: got %v, want %v", decoded.NanoD, original.NanoD)
	}
}

// roundtrip parses the given KDL document string and then emits it back to a
// string, essentially normalizing/formatting it.
func roundtrip(doc string) (string, error) {
	parsed, err := kdl.Parse(strings.NewReader(doc))
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = kdl.Emit(parsed, buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
