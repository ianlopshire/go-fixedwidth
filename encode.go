package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"fmt"
	"io"
	"reflect"
	"strconv"
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
	w               *bufio.Writer
	lineTerminator  []byte
	errorOnOverflow bool
	numericFormat   *fieldFormat
}

var defaultFormat = &fieldFormat{
	rightAlign: false,
	padChar:    byte(' '),
}

// EncoderOption is a constructor option affecting the behavior of the Encoder
type EncoderOption func(enc *Encoder)

// WithRightAlignedZeroPaddedNumbers will encode all ints/uints/floats right-aligned and zero-padded.
func WithRightAlignedZeroPaddedNumbers() EncoderOption {
	return func(enc *Encoder) {
		enc.numericFormat = &fieldFormat{
			rightAlign: true,
			padChar:    byte('0'),
		}
	}
}

func WithOverflowErrors() EncoderOption {
	return func(enc *Encoder) {
		enc.errorOnOverflow = true
	}
}

// NewEncoder returns a new encoder that writes to w, with optional options.
func NewEncoder(w io.Writer, options ...EncoderOption) *Encoder {
	enc := &Encoder{
		w:              bufio.NewWriter(w),
		lineTerminator: []byte("\n"),
	}

	for _, option := range options {
		option(enc)
	}

	return enc
}

// SetLineTerminator sets the character(s) that will be used to terminate lines.
//
// The default value is "\n".
func (e *Encoder) SetLineTerminator(lineTerminator []byte) {
	e.lineTerminator = lineTerminator
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
	b, err := newValueEncoder(v.Type())(v, e)
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

type valueEncoder func(v reflect.Value, enc *Encoder) ([]byte, error)

func newValueEncoder(t reflect.Type) valueEncoder {
	if t == nil {
		return nilEncoder
	}
	if t.Implements(reflect.TypeOf(new(encoding.TextMarshaler)).Elem()) {
		return textMarshalerEncoder
	}

	switch t.Kind() {
	case reflect.Ptr, reflect.Interface:
		return ptrInterfaceEncoder
	case reflect.Struct:
		return structEncoder
	case reflect.String:
		return stringEncoder
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return intEncoder
	case reflect.Float64:
		return floatEncoder(2, 64)
	case reflect.Float32:
		return floatEncoder(2, 32)
	}
	return unknownTypeEncoder(t)
}

func structEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	ss := cachedStructSpec(v.Type())
	dst := bytes.Repeat([]byte(" "), ss.ll)

	for i, spec := range ss.fieldSpecs {
		if !spec.ok {
			continue
		}

		val, err := spec.encoder(v.Field(i), enc)
		if err != nil {
			return nil, err
		}

		var fieldLen = spec.endPos - spec.startPos + 1
		if enc.errorOnOverflow && len(val) > fieldLen {
			return nil, fmt.Errorf("Value '%v' of field %v is too long; %v length where field is only %v wide", string(val), spec.name, len(val), fieldLen)
		}

		// prefer the field's format if it has one, falling back to the encoder's format options, falling back to the default format
		var format = spec.format
		if format == nil && spec.isNumeric && enc.numericFormat != nil {
			format = enc.numericFormat
		}

		if format == nil {
			format = defaultFormat
		}

		var fillStart int

		if format.rightAlign {
			var startPos = spec.startPos + fieldLen - len(val) - 1
			copy(dst[startPos:spec.endPos:spec.endPos], val)
			fillStart = spec.startPos - 1
		} else {
			copy(dst[spec.startPos-1:spec.endPos:spec.endPos], val)
			fillStart = spec.startPos + len(val) - 1
		}

		var fillLength = fieldLen - len(val)
		for i := 0; i < fillLength; i++ {
			dst[fillStart+i] = format.padChar
		}
	}
	return dst, nil
}

func textMarshalerEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	return v.Interface().(encoding.TextMarshaler).MarshalText()
}

func ptrInterfaceEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	if v.IsNil() {
		return nilEncoder(v, enc)
	}
	return newValueEncoder(v.Elem().Type())(v.Elem(), enc)
}

func stringEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	return []byte(v.String()), nil
}

func intEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	return []byte(strconv.Itoa(int(v.Int()))), nil
}

func floatEncoder(perc, bitSize int) valueEncoder {
	return func(v reflect.Value, enc *Encoder) ([]byte, error) {
		return []byte(strconv.FormatFloat(v.Float(), 'f', perc, bitSize)), nil
	}
}

func nilEncoder(v reflect.Value, enc *Encoder) ([]byte, error) {
	return nil, nil
}

func unknownTypeEncoder(t reflect.Type) valueEncoder {
	return func(value reflect.Value, enc *Encoder) ([]byte, error) {
		return nil, &MarshalInvalidTypeError{typeName: t.Name()}
	}
}
