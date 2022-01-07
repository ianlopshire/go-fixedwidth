package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Marshal returns the fixed-width encoding of v.
//
// v must be an encodable type or a slice of an encodable
// type. If v is a slice, each item will be treated as a
// line. If v is a single encodable type, a single line
// will be encoded.
//
// In order for a type to be encodable, it must implement
// the encoding.TextMarshaler interface or be based on one
// of the following builtin types: string, int, int64,
// int32, int16, int8, uint, uint64, uint32, uint16,
// uint8, float64, float32, bool, or struct. Pointers to
// encodable types and interfaces containing encodable
// types are also encodable.
//
// nil pointers and interfaces will be omitted. zero vales
// will be encoded normally.
//
// A struct is encoded to a single slice of bytes. Each
// field in a struct will be encoded and placed at the
// position defined by its struct tags. The tags should be
// formatted as `fixed:"{startPos},{endPos}"`. Positions
// start at 1. The interval is inclusive. Fields without
// tags and Fields of an un-encodable type are ignored.
//
// If the encoded value of a field is longer than the
// length of the position interval, the overflow is
// truncated.
func Marshal(v interface{}) ([]byte, error) {
	buff := bytes.NewBuffer(nil)
	err := NewEncoder(buff).Encode(v)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// MarshalInvalidTypeError describes an invalid type being marshaled.
type MarshalInvalidTypeError struct {
	typeName string
}

func (e *MarshalInvalidTypeError) Error() string {
	return "fixedwidth: cannot marshal unknown Type " + e.typeName
}

// An Encoder writes fixed-width formatted data to an output
// stream.
type Encoder struct {
	w              *bufio.Writer
	lineTerminator []byte

	useCodepointIndices bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:              bufio.NewWriter(w),
		lineTerminator: []byte("\n"),
	}
}

// SetLineTerminator sets the character(s) that will be used to terminate lines.
//
// The default value is "\n".
func (e *Encoder) SetLineTerminator(lineTerminator []byte) {
	e.lineTerminator = lineTerminator
}

// SetUseCodepointIndices configures `Encoder` on whether the indices in the
// `fixedwidth` struct tags are expressed in terms of bytes (the default
// behavior) or in terms of UTF-8 decoded codepoints.
func (e *Encoder) SetUseCodepointIndices(use bool) {
	e.useCodepointIndices = use
}

// Encode writes the fixed-width encoding of v to the
// stream.
// See the documentation for Marshal for details about
// encoding behavior.
func (e *Encoder) Encode(i interface{}) (err error) {
	if i == nil {
		return nil
	}

	// check to see if i should be encoded into multiple lines
	v := reflect.ValueOf(i)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.Slice {
		// encode each slice element to a line
		err = e.writeLines(v)
	} else {
		// this is a single object so encode the original vale to a line
		err = e.writeLine(reflect.ValueOf(i))
	}
	if err != nil {
		return err
	}
	return e.w.Flush()
}

func (e *Encoder) writeLines(v reflect.Value) error {
	for i := 0; i < v.Len(); i++ {
		err := e.writeLine(v.Index(i))
		if err != nil {
			return err
		}

		if i != v.Len()-1 {
			_, err := e.w.Write(e.lineTerminator)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Encoder) writeLine(v reflect.Value) (err error) {
	b, err := newValueEncoder(v.Type(), e.useCodepointIndices)(v)
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(b.data)
	return err
}

type valueEncoder func(v reflect.Value) (rawValue, error)

func newValueEncoder(t reflect.Type, useCodepointIndices bool) valueEncoder {
	if t == nil {
		return nilEncoder
	}
	if t.Implements(reflect.TypeOf(new(encoding.TextMarshaler)).Elem()) {
		return textMarshalerEncoder(useCodepointIndices)
	}

	switch t.Kind() {
	case reflect.Ptr, reflect.Interface:
		return ptrInterfaceEncoder(useCodepointIndices)
	case reflect.Struct:
		return structEncoder(useCodepointIndices)
	case reflect.String:
		return stringEncoder(useCodepointIndices)
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return intEncoder
	case reflect.Float64:
		return floatEncoder(2, 64)
	case reflect.Float32:
		return floatEncoder(2, 32)
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return uintEncoder
	case reflect.Bool:
		return boolEncoder
	}
	return unknownTypeEncoder(t)
}

func (ve valueEncoder) Write(b *lineBuilder, v reflect.Value, spec fieldSpec) error {
	format := spec.format
	startIndex := spec.startPos - 1
	value, err := ve(v)
	if err != nil {
		return err
	}

	if value.len() < spec.len() {
		switch {
		case spec.format.alignment == right:
			padding := strings.Repeat(string(format.padChar), spec.len()-value.len())
			b.WriteASCII(startIndex, padding)
			b.WriteValue(startIndex+len(padding), value)
			return nil

		// The second case in this block is a special case to maintain backward
		// compatibility. In previous versions of the library, only len(value) bytes were
		// written to dst. This means overlapping intervals can, in effect, be used to
		// coalesce a value.
		case format.alignment == left, format.alignment == defaultAlignment && format.padChar != ' ':
			padding := strings.Repeat(string(format.padChar), spec.len()-value.len())

			b.WriteValue(startIndex, value)
			b.WriteASCII(startIndex+value.len(), padding)
			return nil
		}
	}

	if value.len() > spec.len() {
		// If the value is too long it needs to be trimmed.
		// TODO: Add strict mode that returns in this case.
		value, err = value.slice(0, spec.len()-1)
		if err != nil {
			return err
		}
	}

	b.WriteValue(startIndex, value)
	return nil
}

func structEncoder(useCodepointIndices bool) valueEncoder {
	return func(v reflect.Value) (rawValue, error) {
		ss := cachedStructSpec(v.Type())

		// Add a 10% headroom to the builder when codepoint indices are being used.
		c := ss.ll
		if useCodepointIndices {
			c = int(1.1*float64(ss.ll)) + 1
		}
		b := newLineBuilder(ss.ll, c, ' ')

		for i, spec := range ss.fieldSpecs {
			if !spec.ok {
				continue
			}

			enc := spec.getEncoder(useCodepointIndices)
			err := enc.Write(b, v.Field(i), spec)
			if err != nil {
				return rawValue{}, err
			}
		}

		return b.AsRawValue(), nil
	}
}

func textMarshalerEncoder(useCodepointIndices bool) valueEncoder {
	return func(v reflect.Value) (rawValue, error) {
		txt, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return rawValue{}, err
		}
		return newRawValue(string(txt), useCodepointIndices)
	}
}

func ptrInterfaceEncoder(useCodepointIndices bool) valueEncoder {
	return func(v reflect.Value) (rawValue, error) {
		if v.IsNil() {
			return nilEncoder(v)
		}
		return newValueEncoder(v.Elem().Type(), useCodepointIndices)(v.Elem())
	}
}

func stringEncoder(useCodepointIndices bool) valueEncoder {
	return func(v reflect.Value) (rawValue, error) {
		return newRawValue(v.String(), useCodepointIndices)
	}
}
func intEncoder(v reflect.Value) (rawValue, error) {
	return newRawValue(strconv.Itoa(int(v.Int())), false)
}

func floatEncoder(perc, bitSize int) valueEncoder {
	return func(v reflect.Value) (rawValue, error) {
		return newRawValue(strconv.FormatFloat(v.Float(), 'f', perc, bitSize), false)
	}
}

func boolEncoder(v reflect.Value) (rawValue, error) {
	return newRawValue(strconv.FormatBool(v.Bool()), false)
}

func nilEncoder(_ reflect.Value) (rawValue, error) {
	return rawValue{}, nil
}

func unknownTypeEncoder(t reflect.Type) valueEncoder {
	return func(value reflect.Value) (rawValue, error) {
		return rawValue{}, &MarshalInvalidTypeError{typeName: t.Name()}
	}
}

func uintEncoder(v reflect.Value) (rawValue, error) {
	return newRawValue(strconv.FormatUint(v.Uint(), 10), false)
}
