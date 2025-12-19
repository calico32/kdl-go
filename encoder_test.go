package kdl_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
)

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
