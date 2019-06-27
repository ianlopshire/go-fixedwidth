package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"errors"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// Unmarshal parses fixed width encoded data and stores the
// result in the value pointed to by v. If v is nil or not a
// pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(data)).Decode(v)
}

// A Decoder reads and decodes fixed width data from an input stream.
type Decoder struct {
	data                *bufio.Reader
	done                bool
	useCodepointIndices bool
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		data: bufio.NewReader(r),
	}
}

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "fixedwidth: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "fixedwidth: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "fixedwidth: Unmarshal(nil " + e.Type.String() + ")"
}

// An UnmarshalTypeError describes a value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // the raw value
	Type   reflect.Type // type of Go value it could not be assigned to
	Struct string       // name of the struct type containing the field
	Field  string       // name of the field holding the Go value
	Cause  error        // original error
}

func (e *UnmarshalTypeError) Error() string {
	var s string
	if e.Struct != "" || e.Field != "" {
		s = "fixedwidth: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	} else {
		s = "fixedwidth: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
	}
	if e.Cause != nil {
		return s + ":" + e.Cause.Error()
	}
	return s
}

// SetUseCodepointIndices configures `Decoder` on whether the indices in the
// `fixedwidth` struct tags are expressed in terms of bytes (the default
// behavior) or in terms of UTF-8 decoded codepoints.
func (d *Decoder) SetUseCodepointIndices(use bool) {
	d.useCodepointIndices = use
}

// Decode reads from its input and stores the decoded data to the value
// pointed to by v.
//
// In the case that v points to a struct value, Decode will read a
// single line from the input. If there is no data remaining in the file,
// returns io.EOF
//
// In the case that v points to a slice value, Decode will read until
// the end of its input.
func (d *Decoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	if reflect.Indirect(reflect.ValueOf(v)).Kind() == reflect.Slice {
		return d.readLines(reflect.ValueOf(v).Elem())
	}

	err, ok := d.readLine(reflect.ValueOf(v))
	if d.done && err == nil && !ok {
		// d.done means we've reached the end of the file. err == nil && !ok
		// indicates that there was no data to read, so we propagate an io.EOF
		// upwards so our caller knows there is no data left.
		return io.EOF
	}
	return err
}

func (d *Decoder) readLines(v reflect.Value) (err error) {
	ct := v.Type().Elem()
	for {
		nv := reflect.New(ct).Elem()
		err, ok := d.readLine(nv)
		if err != nil {
			return err
		}
		if ok {
			v.Set(reflect.Append(v, nv))
		}
		if d.done {
			break
		}
	}
	return nil
}

type rawLine struct {
	bytes []byte
	// Used when `SetUseCodepointIndices` has been called on `Decoder`. A
	// mapping of codepoint indices into the bytes. So the
	// `codepointIndices[n]` is the starting position for the n-th codepoint in
	// `bytes`.
	codepointIndices []int
}

func newRawLine(bytes []byte, useCodepointIndices bool) (rawLine, error) {
	line := rawLine{
		bytes: bytes,
	}
	if useCodepointIndices {
		bytesIdx := 0
		codepointIdx := 0
		// Lazily allocate this only if the line actaully contains a multi-byte
		// character.
		codepointIndices := []int(nil)
		for bytesIdx < len(bytes) {
			_, codepointSize := utf8.DecodeRune(bytes[bytesIdx:])
			if codepointSize == 0 {
				return rawLine{}, errors.New("Invalid codepoint")
			}
			// We have a multi-byte codepoint, we need to allocate
			// codepointIndices
			if codepointIndices == nil && codepointSize > 1 {
				codepointIndices = []int{}
				for i := 0; i < bytesIdx; i++ {
					codepointIndices = append(codepointIndices, i)
				}
			}
			if codepointIndices != nil {
				codepointIndices = append(codepointIndices, bytesIdx)
			}
			bytesIdx += codepointSize
			codepointIdx += 1
		}
		line.codepointIndices = codepointIndices
	}
	return line, nil
}

func (d *Decoder) readLine(v reflect.Value) (err error, ok bool) {
	var line []byte
	line, err = d.data.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return err, false
	}
	if err == io.EOF {
		d.done = true

		if line == nil || len(line) <= 0 || line[0] == '\n' {
			// skip last empty lines
			return nil, false
		}
	}
	rawLine, err := newRawLine(line, d.useCodepointIndices)
	if err != nil {
		return
	}
	return newValueSetter(v.Type())(v, rawLine), true
}

func rawValueFromLine(line rawLine, startPos, endPos int) rawLine {
	if line.codepointIndices != nil {
		if len(line.codepointIndices) == 0 || startPos > len(line.codepointIndices) {
			return rawLine{bytes: []byte{}}
		}
		if endPos > len(line.codepointIndices) {
			endPos = len(line.codepointIndices)
		}
		relevantIndices := line.codepointIndices[startPos-1 : endPos]
		return rawLine{
			bytes:            bytes.TrimSpace(line.bytes[relevantIndices[0]:relevantIndices[len(relevantIndices)-1]]),
			codepointIndices: relevantIndices,
		}
	} else {
		if len(line.bytes) == 0 || startPos > len(line.bytes) {
			return rawLine{bytes: []byte{}}
		}
		if endPos > len(line.bytes) {
			endPos = len(line.bytes)
		}
		return rawLine{
			bytes: bytes.TrimSpace(line.bytes[startPos-1 : endPos]),
		}
	}
}

type valueSetter func(v reflect.Value, raw rawLine) error

var textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func newValueSetter(t reflect.Type) valueSetter {
	if t.Implements(textUnmarshalerType) {
		return textUnmarshalerSetter(t, false)
	}
	if reflect.PtrTo(t).Implements(textUnmarshalerType) {
		return textUnmarshalerSetter(t, true)
	}

	switch t.Kind() {
	case reflect.Ptr:
		return ptrSetter(t)
	case reflect.Interface:
		return interfaceSetter
	case reflect.Struct:
		return structSetter
	case reflect.String:
		return stringSetter
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return intSetter
	case reflect.Float32:
		return floatSetter(32)
	case reflect.Float64:
		return floatSetter(64)
	}
	return unknownSetter
}

func structSetter(v reflect.Value, raw rawLine) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		if !fv.IsValid() {
			continue
		}
		sf := t.Field(i)
		startPos, endPos, ok := parseTag(sf.Tag.Get("fixed"))
		if !ok {
			continue
		}
		rawValue := rawValueFromLine(raw, startPos, endPos)
		err := newValueSetter(sf.Type)(fv, rawValue)
		if err != nil {
			return &UnmarshalTypeError{string(rawValue.bytes), sf.Type, t.Name(), sf.Name, err}
		}
	}
	return nil
}

func unknownSetter(v reflect.Value, raw rawLine) error {
	return errors.New("fixedwidth: unknown type")
}

func nilSetter(v reflect.Value, _ rawLine) error {
	v.Set(reflect.Zero(v.Type()))
	return nil
}

func textUnmarshalerSetter(t reflect.Type, shouldAddr bool) valueSetter {
	return func(v reflect.Value, raw rawLine) error {
		if shouldAddr {
			v = v.Addr()
		}
		// set to zero value if this is nil
		if t.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return v.Interface().(encoding.TextUnmarshaler).UnmarshalText(raw.bytes)
	}
}

func interfaceSetter(v reflect.Value, raw rawLine) error {
	return newValueSetter(v.Elem().Type())(v.Elem(), raw)
}

func ptrSetter(t reflect.Type) valueSetter {
	return func(v reflect.Value, raw rawLine) error {
		if len(raw.bytes) <= 0 {
			return nilSetter(v, raw)
		}
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return newValueSetter(v.Elem().Type())(reflect.Indirect(v), raw)
	}
}

func stringSetter(v reflect.Value, raw rawLine) error {
	v.SetString(string(raw.bytes))
	return nil
}

func intSetter(v reflect.Value, raw rawLine) error {
	if len(raw.bytes) < 1 {
		return nil
	}
	i, err := strconv.Atoi(string(raw.bytes))
	if err != nil {
		return err
	}
	v.SetInt(int64(i))
	return nil
}

func floatSetter(bitSize int) valueSetter {
	return func(v reflect.Value, raw rawLine) error {
		if len(raw.bytes) < 1 {
			return nil
		}
		f, err := strconv.ParseFloat(string(raw.bytes), bitSize)
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	}
}
