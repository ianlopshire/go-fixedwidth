package fixedwidth

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

func ExampleMarshal() {
	// define some data to encode
	people := []struct {
		ID        int     `fixed:"1,5"`
		FirstName string  `fixed:"6,15"`
		LastName  string  `fixed:"16,25"`
		Grade     float64 `fixed:"26,30"`
		Age       uint    `fixed:"31,33"`
		Alive     bool    `fixed:"34,39"`
	}{
		{1, "Ian", "Lopshire", 99.5, 20, true},
	}

	data, err := Marshal(people)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", data)
	// Output:
	// 1    Ian       Lopshire  99.5020 true
}

func ExampleMarshal_configurableFormatting() {
	// define some data to encode
	people := []struct {
		ID        int     `fixed:"1,5,right,#"`
		FirstName string  `fixed:"6,15,right,#"`
		LastName  string  `fixed:"16,25,right,#"`
		Grade     float64 `fixed:"26,30,right,#"`
		Age       uint    `fixed:"31,33,right,#"`
		Alive     bool    `fixed:"34,39,right,#"`
	}{
		{1, "Ian", "Lopshire", 99.5, 20, true},
	}

	data, err := Marshal(people)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", data)
	// Output:
	// ####1#######Ian##Lopshire99.50#20##true
}

func TestMarshal(t *testing.T) {
	type H struct {
		F1 interface{} `fixed:"1,5"`
		F2 interface{} `fixed:"6,10"`
	}
	type H2 struct {
		F1 bool `fixed:"1,1"`
	}
	tagHelper := struct {
		Valid       string `fixed:"1,5"`
		NoTags      string
		InvalidTags string `fixed:"5"`
	}{"foo", "foo", "foo"}
	marshalError := errors.New("marshal error")

	var invtype func()

	for _, tt := range []struct {
		name      string
		i         interface{}
		o         []byte
		shouldErr bool
	}{
		{"single line", H{"foo", 1}, []byte("foo  1    "), false},
		{"multiple line", []H{{"foo", 1}, {"bar", 2}}, []byte("foo  1    \nbar  2    "), false},
		{"multiple line (diff struct)", []interface{}{H{"foo", 1}, H2{false}}, []byte("foo  1    \nf"), false},
		{"empty slice", []H{}, nil, false},
		{"pointer", &H{"foo", 1}, []byte("foo  1    "), false},
		{"nil", nil, nil, false},
		{"invalid type", invtype, nil, true},
		{"invalid type in struct", H{"foo", invtype}, nil, true},
		{"marshal error", EncodableString{"", marshalError}, nil, true},
		{"invalid tags", tagHelper, []byte("foo  "), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			o, err := Marshal(tt.i)
			if tt.shouldErr != (err != nil) {
				t.Errorf("Marshal() shouldErr expected %v, have %v (%v)", tt.shouldErr, err != nil, err)
			}
			if !tt.shouldErr && !bytes.Equal(o, tt.o) {
				t.Errorf("Marshal() expected %q, have %q", string(tt.o), string(o))
			}

			// All tests should also pass with codepoint indices enabled.
			t.Run("use codepoint indices", func(t *testing.T) {
				buff := bytes.NewBuffer(nil)
				enc := NewEncoder(buff)
				enc.SetUseCodepointIndices(true)
				err := enc.Encode(tt.i)
				if tt.shouldErr != (err != nil) {
					t.Errorf("Marshal() shouldErr expected %v, have %v (%v)", tt.shouldErr, err != nil, err)
				}
				if !tt.shouldErr && !bytes.Equal(buff.Bytes(), tt.o) {
					t.Errorf("Marshal() expected %q, have %q", string(tt.o), string(o))
				}
			})

		})
	}
}

func TestMarshal_useCodepointIndices(t *testing.T) {
	type H struct {
		F1 string `fixed:"1,5"`
		F2 string `fixed:"6,10"`
		F3 string `fixed:"11,15"`
	}

	type HF struct {
		F1 string `fixed:"1,5,right,#"`
		F2 string `fixed:"6,10,left,#"`
		F3 string `fixed:"11,15"`
	}

	for _, tt := range []struct {
		name string
		i    interface{}
		o    []byte
	}{
		{name: "base case", i: H{"føø", "bår", "båz"}, o: []byte(`føø  bår  båz  `)},
		{name: "overflow", i: H{"føøøøøøøøøø", "bååååååååår", "bååååååååz"}, o: []byte(`føøøøbååååbåååå`)},
		{name: "formatted", i: HF{"føø", "bår", "båz"}, o: []byte(`##føøbår##båz  `)},
		{name: "multibformatted overflow", i: HF{"føøøøøøøøøø", "bååååååååår", "bååååååååz"}, o: []byte(`føøøøbååååbåååå`)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buff := bytes.NewBuffer(nil)
			enc := NewEncoder(buff)
			enc.SetUseCodepointIndices(true)
			if err := enc.Encode(tt.i); err != nil {
				t.Errorf("Marshal() unexpected error:  %v", err)
				return
			}
			if o := buff.Bytes(); !bytes.Equal(o, tt.o) {
				t.Errorf("Marshal() expected %s, have %s", tt.o, o)
			}
		})
	}
}

func TestMarshal_format(t *testing.T) {
	type H struct {
		F1 string `fixed:"1,5,left"`
		F2 string `fixed:"6,10,left,#"`
		F3 string `fixed:"11,15,right"`
		F4 string `fixed:"16,20,right,#"`
		F5 string `fixed:"21,25,default"`
		F6 string `fixed:"26,30,default,#"`
	}

	for _, tt := range []struct {
		name      string
		v         interface{}
		want      []byte
		shouldErr bool
	}{
		{
			name:      "base case",
			v:         H{"foo", "bar", "biz", "baz", "bor", "box"},
			want:      []byte(`foo  ` + `bar##` + `  biz` + `##baz` + `bor  ` + `box##`),
			shouldErr: false,
		},
		{
			name:      "empty",
			v:         H{"", "", "", "", "", ""},
			want:      []byte(`     ` + `#####` + `     ` + `#####` + `     ` + `#####`),
			shouldErr: false,
		},
		{
			name:      "overflow",
			v:         H{"12345678", "12345678", "12345678", "12345678", "12345678", "12345678"},
			want:      []byte(`12345` + `12345` + `12345` + `12345` + `12345` + `12345`),
			shouldErr: false,
		},
		{
			name: "pad right",
			v: struct {
				F1 string `fixed:"1,5,right,#"`
			}{"foo"},
			want:      []byte(`##foo`),
			shouldErr: false,
		},
		{
			name: "pad left",
			v: struct {
				F1 string `fixed:"1,5,left,#"`
			}{"foo"},
			want:      []byte(`foo##`),
			shouldErr: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			have, err := Marshal(tt.v)
			if tt.shouldErr != (err != nil) {
				t.Errorf("Marshal() err want %v, have %v (%v)", tt.shouldErr, err != nil, err)
			}
			if !bytes.Equal(tt.want, have) {
				t.Errorf("Marshal() want %q, have %q", string(tt.want), string(have))
			}
		})
	}
}

func TestMarshal_backwardCompatibility(t *testing.T) {
	// Overlapping intervals can, in effect, be used to coalesce a value. This tests
	// ensures this special does not break.
	t.Run("interval overlap coalesce", func(t *testing.T) {
		type H struct {
			F1 string `fixed:"1,5"`
			F2 string `fixed:"1,5"`
		}

		have, err := Marshal(H{F1: "val"})
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		if want := []byte(`val  `); !bytes.Equal(have, want) {
			t.Errorf("Marshal() want %q, have %q", string(want), string(have))
		}

		have, err = Marshal(H{F2: "val"})
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		if want := []byte(`val  `); !bytes.Equal(have, want) {
			t.Errorf("Marshal() want %q, have %q", string(want), string(have))
		}
	})
}

func TestNewValueEncoder(t *testing.T) {
	for _, tt := range []struct {
		name      string
		i         interface{}
		o         []byte
		shouldErr bool
	}{
		{"nil", nil, []byte(""), false},
		{"nil interface", interface{}(nil), []byte(""), false},

		{"[]string (invalid)", []string{"a", "b"}, []byte(""), true},
		{"[]string interface (invalid)", interface{}([]string{"a", "b"}), []byte(""), true},

		{"string", "foo", []byte("foo"), false},
		{"string interface", interface{}("foo"), []byte("foo"), false},
		{"string empty", "", []byte(""), false},
		{"*string", stringp("foo"), []byte("foo"), false},
		{"*string empty", stringp(""), []byte(""), false},
		{"*string nil", nilString, []byte(""), false},

		{"float64", float64(123.4567), []byte("123.46"), false},
		{"float64 interface", interface{}(float64(123.4567)), []byte("123.46"), false},
		{"float64 zero", float64(0), []byte("0.00"), false},
		{"*float64", float64p(123.4567), []byte("123.46"), false},
		{"*float64 zero", float64p(0), []byte("0.00"), false},
		{"*float64 nil", nilFloat64, []byte(""), false},

		{"float32", float32(123.4567), []byte("123.46"), false},
		{"float32 interface", interface{}(float32(123.4567)), []byte("123.46"), false},
		{"float32 zero", float32(0), []byte("0.00"), false},
		{"*float32", float32p(123.4567), []byte("123.46"), false},
		{"*float32 zero", float32p(0), []byte("0.00"), false},
		{"*float32 nil", nilFloat32, []byte(""), false},

		{"int", int(123), []byte("123"), false},
		{"int interface", interface{}(int(123)), []byte("123"), false},
		{"int zero", int(0), []byte("0"), false},
		{"*int", intp(123), []byte("123"), false},
		{"*int zero", intp(0), []byte("0"), false},
		{"*int nil", nilInt, []byte(""), false},

		{"bool positive", bool(true), []byte("true"), false},
		{"bool interface positive", interface{}(bool(true)), []byte("true"), false},
		{"*bool positive", boolp(true), []byte("true"), false},
		{"*bool negative", boolp(false), []byte("false"), false},

		{"TextUnmarshaler", EncodableString{"foo", nil}, []byte("foo"), false},
		{"TextUnmarshaler interface", interface{}(EncodableString{"foo", nil}), []byte("foo"), false},
		{"TextUnmarshaler error", EncodableString{"foo", errors.New("TextUnmarshaler error")}, []byte("foo"), true},

		{"uint", uint(123), []byte("123"), false},
		{"uint interface", interface{}(uint(123)), []byte("123"), false},
		{"uint zero", uint(0), []byte("0"), false},
		{"*uint", uintp(123), []byte("123"), false},
		{"*uint zero", uintp(0), []byte("0"), false},
		{"*uint nil", nilUint, []byte(""), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			o, err := newValueEncoder(reflect.TypeOf(tt.i), false)(reflect.ValueOf(tt.i))
			if tt.shouldErr != (err != nil) {
				t.Errorf("newValueEncoder(%s)() shouldErr expected %v, have %v (%v)", reflect.TypeOf(tt.i).Name(), tt.shouldErr, err != nil, err)
			}
			if !tt.shouldErr && !bytes.Equal([]byte(o.data), tt.o) {
				t.Errorf("newValueEncoder(%s)() expected %v, have %v", reflect.TypeOf(tt.i).Name(), tt.o, o)
			}
		})
	}
}

func TestEncoder_SetLineTerminator(t *testing.T) {
	buff := new(bytes.Buffer)
	enc := NewEncoder(buff)
	enc.SetLineTerminator([]byte{'\r', '\n'})

	input := []interface{}{
		EncodableString{"foo", nil},
		EncodableString{"bar", nil},
	}

	err := enc.Encode(input)
	if err != nil {
		t.Fatal("Encode() unexpected error")
	}

	expected := []byte("foo\r\nbar")
	if !bytes.Equal(expected, buff.Bytes()) {
		t.Errorf("Encode() expected %q, have %q", expected, buff.Bytes())
	}
}
