package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"io"
	"reflect"
	"strconv"
)

// ValueWriter is responsible for writing an encoded value
// the destination. ValueWriter should handle padding and
// truncation.
//
// The value and destination params are provided by the encoder
// The destination param will always have the length and capacity
// of the interval being written. by default the the destination
// is filled with spaces.
type ValueWriter func(value, destination []byte) error

// PadRight is a ValueWriter that pads values on the right.
// If the the value is longer than the destination, the value
// will be truncated on the right.
func PadRight(value, destination []byte) error {
	for i := 0; i < len(value) && i < len(destination); i++ {
		destination[i] = value[i]
	}
	return nil
}

// PadLeft is a ValueWriter that pads values on the left.
// If the the value is longer than the destination, the value
// will be truncated on the left.
func PadLeft(value, destination []byte) error {
	for i := 0; i < len(value) && i < len(destination); i++ {
		destination[len(destination)-i-1] = value[len(value)-i-1]
	}
	return nil
}

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
// int32, int16, int8, float64, float32, or struct.
// Pointers to encodable types and interfaces containing
// encodable types are also encodable.
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

func MarshalWith(v interface{}, vw ValueWriter) ([]byte, error) {
	buff := bytes.NewBuffer(nil)
	err := NewEncoderWith(buff, vw).Encode(v)
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
	w  *bufio.Writer
	vw ValueWriter
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		bufio.NewWriter(w),
		PadRight,
	}
}

func NewEncoderWith(w io.Writer, vw ValueWriter) *Encoder {
	return &Encoder{
		bufio.NewWriter(w),
		vw,
	}
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
			_, err := e.w.Write([]byte("\n"))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Encoder) writeLine(v reflect.Value) (err error) {
	b, err := e.newValueEncoder(v.Type())(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

type valueEncoder func(v reflect.Value) ([]byte, error)

func (e *Encoder) newValueEncoder(t reflect.Type) valueEncoder {
	if t == nil {
		return e.nilEncoder
	}
	if t.Implements(reflect.TypeOf(new(encoding.TextMarshaler)).Elem()) {
		return e.textMarshalerEncoder
	}

	switch t.Kind() {
	case reflect.Ptr, reflect.Interface:
		return e.ptrInterfaceEncoder
	case reflect.Struct:
		return e.structEncoder
	case reflect.String:
		return e.stringEncoder
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return e.intEncoder
	case reflect.Float64:
		return e.floatEncoder(2, 64)
	case reflect.Float32:
		return e.floatEncoder(2, 32)
	}
	return e.unknownTypeEncoder(t)
}

func (e *Encoder) structEncoder(v reflect.Value) ([]byte, error) {
	var specs []fieldSpec
	for i := 0; i < v.Type().NumField(); i++ {
		f := v.Type().Field(i)
		var (
			err  error
			spec fieldSpec
			ok   bool
		)
		spec.startPos, spec.endPos, ok = parseTag(f.Tag.Get("fixed"))
		if !ok {
			continue
		}
		spec.value, err = e.newValueEncoder(f.Type)(v.Field(i))
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return encodeSpecs(specs, e.vw)
}

type fieldSpec struct {
	startPos, endPos int
	value            []byte
}

func encodeSpecs(specs []fieldSpec, w ValueWriter) ([]byte, error) {
	var ll int
	for _, spec := range specs {
		if spec.endPos > ll {
			ll = spec.endPos
		}
	}
	data := bytes.Repeat([]byte(" "), ll)
	for _, spec := range specs {
		err := w(spec.value, data[spec.startPos-1:spec.endPos:spec.endPos])
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (e *Encoder) textMarshalerEncoder(v reflect.Value) ([]byte, error) {
	return v.Interface().(encoding.TextMarshaler).MarshalText()
}

func (e *Encoder) ptrInterfaceEncoder(v reflect.Value) ([]byte, error) {
	if v.IsNil() {
		return e.nilEncoder(v)
	}
	return e.newValueEncoder(v.Elem().Type())(v.Elem())
}

func (e *Encoder) stringEncoder(v reflect.Value) ([]byte, error) {
	return []byte(v.String()), nil
}

func (e *Encoder) intEncoder(v reflect.Value) ([]byte, error) {
	return []byte(strconv.Itoa(int(v.Int()))), nil
}

func (e *Encoder) floatEncoder(perc, bitSize int) valueEncoder {
	return func(v reflect.Value) ([]byte, error) {
		return []byte(strconv.FormatFloat(v.Float(), 'f', perc, bitSize)), nil
	}
}

func (e *Encoder) nilEncoder(v reflect.Value) ([]byte, error) {
	return nil, nil
}

func (e *Encoder) unknownTypeEncoder(t reflect.Type) valueEncoder {
	return func(value reflect.Value) ([]byte, error) {
		return nil, &MarshalInvalidTypeError{typeName: t.Name()}
	}
}
