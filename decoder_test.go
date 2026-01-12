package kdl_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
	"github.com/davecgh/go-spew/spew"
)

type DecoderTest struct {
	Document string
	Expected any
}

type Person struct {
	Name  string            `kdl:"name"`
	Age   int               `kdl:"age"`
	Props map[string]string `kdl:",properties,children"`
	Bool  bool              `kdl:"bool,presence"`
}

type Value struct {
	Data any `kdl:"data"`
}

type NoTags struct {
	Name string
	Age  int
}

type Personer interface {
	Person() Person
}

func (p *Person) Person() Person { return *p }

type People struct {
	Persons []Person `kdl:"person,multiple"`
}

type PeopleInterface struct {
	Persons []*Personer `kdl:"person,multiple"`
}

type Integers struct {
	Int8   int8   `kdl:"int8"`
	Int16  int16  `kdl:"int16"`
	Int32  int32  `kdl:"int32"`
	Int64  int64  `kdl:"int64"`
	Uint8  uint8  `kdl:"uint8"`
	Uint16 uint16 `kdl:"uint16"`
	Uint32 uint32 `kdl:"uint32"`
	Uint64 uint64 `kdl:"uint64"`
}

type IntegerMapKeys struct {
	Int8   map[int8]string   `kdl:"int8"`
	Int16  map[int16]string  `kdl:"int16"`
	Int32  map[int32]string  `kdl:"int32"`
	Int64  map[int64]string  `kdl:"int64"`
	Uint8  map[uint8]string  `kdl:"uint8"`
	Uint16 map[uint16]string `kdl:"uint16"`
	Uint32 map[uint32]string `kdl:"uint32"`
	Uint64 map[uint64]string `kdl:"uint64"`
}

type Arguments struct {
	First  string   `kdl:",argument"`
	Second *float64 `kdl:",argument"`
	Rest   []int    `kdl:",arguments"`
}

type ArgumentsNode struct {
	Arguments *Arguments `kdl:"arguments"`
}

type Pointers struct {
	One   *int   `kdl:"one"`
	Two   **int  `kdl:"two"`
	Three ***int `kdl:"three"`
}

var decoderTests = []DecoderTest{
	{"name Alice\nage 30\n", &Person{Name: "Alice", Age: 30}},
	{
		`person {
			name Alice
			age 30
		}
		person {
			name Bob
			age 25
			bool
		}
		person {
			name Charlie
			age 30
			bool #false
		}
		`,
		&People{Persons: []Person{
			{Name: "Alice", Age: 30, Bool: false, Props: map[string]string{}},
			{Name: "Bob", Age: 25, Bool: true, Props: map[string]string{}},
			{Name: "Charlie", Age: 30, Bool: false, Props: map[string]string{}},
		}},
	},
	{
		`person name=Alice age=30 extra=bar
		 person name=Bob age="25" other=baz`,
		&People{Persons: []Person{
			{Name: "Alice", Age: 30, Props: map[string]string{"extra": "bar"}},
			{Name: "Bob", Age: 25, Props: map[string]string{"other": "baz"}},
		}},
	},
	{"int8 42\n", &Integers{Int8: 42}},
	{"int16 42\n", &Integers{Int16: 42}},
	{"int32 42\n", &Integers{Int32: 42}},
	{"int64 42\n", &Integers{Int64: 42}},
	{"uint8 42\n", &Integers{Uint8: 42}},
	{"uint16 42\n", &Integers{Uint16: 42}},
	{"uint32 42\n", &Integers{Uint32: 42}},
	{"uint64 42\n", &Integers{Uint64: 42}},
	{"int8 \"foo\"\n", &IntegerMapKeys{Int8: map[int8]string{0: "foo"}}},
	{"int16 \"foo\"\n", &IntegerMapKeys{Int16: map[int16]string{0: "foo"}}},
	{"int32 \"foo\"\n", &IntegerMapKeys{Int32: map[int32]string{0: "foo"}}},
	{"int64 \"foo\"\n", &IntegerMapKeys{Int64: map[int64]string{0: "foo"}}},
	{"uint8 \"foo\"\n", &IntegerMapKeys{Uint8: map[uint8]string{0: "foo"}}},
	{"uint16 \"foo\"\n", &IntegerMapKeys{Uint16: map[uint16]string{0: "foo"}}},
	{"uint32 \"foo\"\n", &IntegerMapKeys{Uint32: map[uint32]string{0: "foo"}}},
	{"uint64 \"foo\"\n", &IntegerMapKeys{Uint64: map[uint64]string{0: "foo"}}},
	{"arguments Alice 3.14 1 2 3 4 5\n", &ArgumentsNode{Arguments: &Arguments{
		First:  "Alice",
		Second: func() *float64 { v := 3.14; return &v }(),
		Rest:   []int{1, 2, 3, 4, 5},
	}}},
	{"data 42\n", &Value{Data: 42}},
	{"Name Alice\nAge 30\n", &NoTags{Name: "Alice", Age: 30}},
	{"one 1\ntwo 2\nthree 3\n", &Pointers{
		One:   func() *int { v := 1; return &v }(),
		Two:   func() **int { v := 2; p := &v; return &p }(),
		Three: func() ***int { v := 3; p1 := &v; p2 := &p1; return &p2 }(),
	}},
}

func TestDecode(t *testing.T) {
	for i, test := range decoderTests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {

			expected := reflect.ValueOf(test.Expected)
			targetType := expected.Type().Elem()
			actual := reflect.New(targetType)

			err := kdl.Decode(strings.NewReader(test.Document), actual.Elem())
			if err != nil {
				t.Errorf("Decode failed: %+v", err)
				return
			}
			if !reflect.DeepEqual(expected.Elem().Interface(), actual.Elem().Interface()) {
				t.Errorf("Value mismatch\nExpected: %s\nGot: %s", spew.Sdump(expected.Elem().Interface()), spew.Sdump(actual.Elem().Interface()))
			}
		})
	}
}

func TestDecodeEmptyInterface(t *testing.T) {
	doc := `
		name Alice
		age 30
		location {
			city Wonderland
			country Fiction
		}
	`

	var actual any
	err := kdl.Decode(strings.NewReader(doc), &actual)
	if err != nil {
		t.Error(err)
		return
	}

	expected := map[string]any{
		"name": "Alice",
		"age":  30,
		"location": map[string]any{
			"city":    "Wonderland",
			"country": "Fiction",
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Value mismatch\nExpected:\n%#v\n\nGot:\n%#v", expected, actual)
	}
}

func TestDecodeInterface(t *testing.T) {
	doc := `
		name Alice
		age 30
	`

	var actual Personer = &Person{}
	err := kdl.Decode(strings.NewReader(doc), &actual)
	if err != nil {
		t.Error(err)
		return
	}

	expected := Person{Name: "Alice", Age: 30}

	if !reflect.DeepEqual(expected, actual.Person()) {
		t.Errorf("Value mismatch\nExpected:\n%#v\n\nGot:\n%#v", expected, actual.Person())
	}
}

type StrictModeTest struct {
	ShouldSucceed bool
	Document      string
	Type          any
}

var strictModeTests = []StrictModeTest{
	{
		true,
		"age 30\n",
		&struct {
			Age int `kdl:"age"`
		}{},
	},
	{
		false,
		"age 30\nextra 42\n",
		&struct {
			Age int `kdl:"age"`
		}{},
	},
	{
		false,
		"age \"30\"\n",
		&struct {
			Age int `kdl:"age"`
		}{},
	},
	{
		true,
		"",
		&struct{}{},
	},
	{
		false,
		"foo bar",
		&struct{}{},
	},
	{
		false,
		"",
		&struct {
			Age int `kdl:"age"`
		}{},
	},
	{
		false,
		"age 30.0\n",
		&struct {
			Age int `kdl:"age"`
		}{},
	},
	{
		false,
		"foo bar baz qux\n",
		&struct {
			Foo []string `kdl:",argument"`
		}{},
	},
	{
		true,
		"foo bar baz qux\n",
		&struct {
			Foo struct {
				Args []string `kdl:",arguments"`
			} `kdl:"foo"`
		}{},
	},
}

func TestDecodeStrictMode(t *testing.T) {
	for i, test := range strictModeTests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {

			expected := reflect.ValueOf(test.Type)
			targetType := expected.Type().Elem()
			actual := reflect.New(targetType)

			err := kdl.DecodeStrict(strings.NewReader(test.Document), actual.Elem())
			if test.ShouldSucceed {
				if err != nil {
					t.Errorf("Unexpected error: %+v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			}
		})
	}
}

func TestDecodeStructTags(t *testing.T) {
	type T struct {
		Arg      int               `kdl:",arg,strict"`
		Args     []int             `kdl:",args"`
		Prop     string            `kdl:"prop,prop"`
		Props    map[string]string `kdl:",props"`
		Child    []string          `kdl:"child,child,multiple"`
		Children map[string]string `kdl:",children"`
	}

	type D struct {
		Node []T `kdl:"node,multiple"`
	}

	doc := `
		node 1 2 "3" 4 prop=foo extra=bar {
			child child1
			child child2
			unmapped1 alice
			unmapped2 bob
		}
		node 5 6 7 "8" prop=bar another=baz {
			child child3
			child child4
			unmapped3 charlie
			unmapped4 dave
		}
	`

	expected := D{
		Node: []T{
			{
				Arg:      1,
				Args:     []int{2, 3, 4},
				Prop:     "foo",
				Props:    map[string]string{"extra": "bar"},
				Child:    []string{"child1", "child2"},
				Children: map[string]string{"unmapped1": "alice", "unmapped2": "bob"},
			},
			{
				Arg:      5,
				Args:     []int{6, 7, 8},
				Prop:     "bar",
				Props:    map[string]string{"another": "baz"},
				Child:    []string{"child3", "child4"},
				Children: map[string]string{"unmapped3": "charlie", "unmapped4": "dave"},
			},
		},
	}

	var actual D
	err := kdl.Decode(strings.NewReader(doc), &actual)
	if err != nil {
		t.Errorf("Decode failed: %+v", err)
		return
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Value mismatch\nExpected:\n%s\nGot:\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}

	empty := `
		node
	`

	var actualEmpty D
	err = kdl.Decode(strings.NewReader(empty), &actualEmpty)
	if err == nil {
		t.Errorf("Expected error for missing required fields, but got none")
		return
	}

	wrongType := `
		node "1"
	`
	var actualWrongType D
	err = kdl.Decode(strings.NewReader(wrongType), &actualWrongType)
	if err == nil {
		t.Errorf("Expected error for wrong argument type, but got none")
		return
	}
}

func TestDecodeStructTagErrors(t *testing.T) {
	tags := []reflect.StructTag{
		`kdl:",arg,arg"`,
		`kdl:"foo,prop,prop"`,
		`kdl:"name,arg"`,
		`kdl:",child"`,
		`kdl:",prop"`,
		`kdl:"foo,arg,args"`,
		`kdl:"bar,prop,props"`,
		`kdl:"baz,child,children"`,
		`kdl:",args,props"`,
		`kdl:",args,children"`,
		`kdl:",arg,multiple"`,
		`kdl:"prop,prop,multiple"`,
	}

	for _, tag := range tags {
		typ := reflect.StructOf([]reflect.StructField{
			{Name: "Field", Type: reflect.TypeFor[string](), Tag: tag},
		})
		ptr := reflect.New(typ).Interface()
		err := kdl.Decode(strings.NewReader(""), ptr)
		if err == nil {
			t.Errorf("Expected error for illegal tag %s, but got none", tag)
			return
		}
	}
}
