package fixedwidth_test

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/ianlopshire/go-fixedwidth"
)

func ExampleUnmarshal() {
	// Define some fixed-with data to parse
	data := []byte("" +
		"1         Ian                 Lopshire" + "\n" +
		"2         John                Doe" + "\n" +
		"3         Jane                Doe" + "\n")

	// Define the format as a struct.
	// The fixed start and end position are defined via struct tags: `fixed:"{startPos},{endPos}"`.
	// Positions start at 1. The interval is inclusive.
	var people []struct {
		ID        int    `fixed:"1,10"`
		FirstName string `fixed:"11,30"`
		LastName  string `fixed:"31,50"`
	}

	err := fixedwidth.Unmarshal(data, &people)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", people[0])
	fmt.Printf("%+v\n", people[1])
	fmt.Printf("%+v\n", people[2])
	// Output:
	// {ID:1 FirstName:Ian LastName:Lopshire}
	// {ID:2 FirstName:John LastName:Doe}
	// {ID:3 FirstName:Jane LastName:Doe}
}

// str is a string that implements the encoding.TextUnmarshaler interface.
// This is useful for testing.
type str string

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *str) UnmarshalText(text []byte) error {
	*s = str(text)
	return nil
}

func TestUnmarshal(t *testing.T) {
	// allTypes contains a field with all current supported types.
	type allTypes struct {
		String          string  `fixed:"1,5"`
		Int             int     `fixed:"6,10"`
		Float           float64 `fixed:"11,15"`
		TextUnmarshaler str     `fixed:"16,20"` // test encoding.TextUnmarshaler functionality
	}
	for _, tt := range []struct {
		name      string
		rawValue  []byte
		target    interface{}
		expected  interface{}
		shouldErr bool
	}{
		{
			name:     "Basic Slice Case",
			rawValue: []byte("foo  123  1.2  bar" + "\n" + "bar  321  2.1  foo"),
			target:   &[]allTypes{},
			expected: &[]allTypes{
				{"foo", 123, 1.2, "bar"},
				{"bar", 321, 2.1, "foo"},
			},
			shouldErr: false,
		},
		{
			name:      "Basic Struct Case",
			rawValue:  []byte("foo  123  1.2  bar"),
			target:    &allTypes{},
			expected:  &allTypes{"foo", 123, 1.2, "bar"},
			shouldErr: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := fixedwidth.Unmarshal(tt.rawValue, tt.target)
			if tt.shouldErr != (err != nil) {
				t.Errorf("Unmarshal() err want %v, have %v (%v)", tt.shouldErr, err != nil, err)
			}
			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("Unmarshal() want %+v, have %+v", tt.target, tt.expected)
			}

		})
	}

	t.Run("Invalid Unmarshal Errors", func(t *testing.T) {
		for _, tt := range []struct {
			name      string
			v         interface{}
			shouldErr bool
		}{
			{"Invalid Unmarshal Nil", nil, true},
			{"Invalid Unmarshal Not Pointer 1", struct{}{}, true},
			{"Invalid Unmarshal Not Pointer 2", []struct{}{}, true},
			{"Valid Unmarshal slice", &[]struct{}{}, false},
			{"Valid Unmarshal struct", &struct{}{}, false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				err := fixedwidth.Unmarshal([]byte{}, tt.v)
				if tt.shouldErr != (err != nil) {
					t.Errorf("Unmarshal() err want %v, have %v (%v)", tt.shouldErr, err != nil, err)
				}
			})
		}
	})
}
