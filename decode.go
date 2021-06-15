package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
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
	scanner             *bufio.Scanner
	lineTerminator      []byte
	done                bool
	useCodepointIndices bool

	lastType       reflect.Type
	lastValuSetter valueSetter
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	dec := &Decoder{
		scanner:        bufio.NewScanner(r),
		lineTerminator: []byte("\n"),
	}
	dec.scanner.Split(dec.scan)
	return dec
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

	if rv.Elem().Kind() == reflect.Slice {
		return d.readLines(rv.Elem())
	}

	err, ok := d.readLine(rv)
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

type rawValue struct {
	data string
	// Used when `SetUseCodepointIndices` has been called on `Decoder`. A
	// mapping of codepoint indices into the bytes. So the
	// `codepointIndices[n]` is the starting position for the n-th codepoint in
	// `bytes`.
	codepointIndices []int
}

func newRawValue(data string, useCodepointIndices bool) (rawValue, error) {
	value := rawValue{
		data: data,
	}
	if useCodepointIndices {
		bytesIdx := findFirstMultiByteChar(data)
		// If we've got multi-byte characters, fill in the rest of codepointIndices.
		if bytesIdx < len(data) {
			codepointIndices := make([]int, bytesIdx)
			for i := 0; i < bytesIdx; i++ {
				codepointIndices[i] = i
			}
			for bytesIdx < len(data) {
				_, codepointSize := utf8.DecodeRuneInString(data[bytesIdx:])
				if codepointSize == 0 {
					return rawValue{}, errors.New("fixedwidth: Invalid codepoint")
				}
				codepointIndices = append(codepointIndices, bytesIdx)
				bytesIdx += codepointSize
			}
			value.codepointIndices = codepointIndices
		}
	}
	return value, nil
}

// Scans bytes, looking for multi-byte characters, returns either the index of
// the first multi-byte chracter or the length of the string if there are none.
func findFirstMultiByteChar(data string) int {
	for i := 0; i < len(data); i++ {
		// We have a multi-byte codepoint, we need to allocate
		// codepointIndices
		if data[i]&0x80 == 0x80 {
			return i
		}
	}
	return len(data)
}

// SetLineTerminator sets the character(s) that will be used to terminate lines.
//
// The default value is "\n".
func (d *Decoder) SetLineTerminator(lineTerminator []byte) {
	if len(lineTerminator) > 0 {
		d.lineTerminator = lineTerminator
	}
}

func (d *Decoder) scan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, d.lineTerminator); i >= 0 {
		// We have a full newline-terminated line.
		return i + len(d.lineTerminator), data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (d *Decoder) readLine(v reflect.Value) (err error, ok bool) {
	ok = d.scanner.Scan()
	if !ok {
		d.done = true
		return nil, false
	}

	line := string(d.scanner.Bytes())

	rawValue, err := newRawValue(line, d.useCodepointIndices)
	if err != nil {
		return
	}
	t := v.Type()
	if t == d.lastType {
		return d.lastValuSetter(v, rawValue), true
	}
	valueSetter := newValueSetter(t)
	d.lastType = t
	d.lastValuSetter = valueSetter
	return valueSetter(v, rawValue), true
}

func rawValueFromLine(value rawValue, startPos, endPos int, format format) rawValue {
	var trimFunc func(string) string

	switch format.alignment {
	case left:
		trimFunc = func(s string) string {
			return strings.TrimRight(s, string(format.padChar))
		}
	case right:
		trimFunc = func(s string) string {
			return strings.TrimLeft(s, string(format.padChar))
		}
	default:
		trimFunc = func(s string) string {
			return strings.Trim(s, string(format.padChar))
		}
	}

	if value.codepointIndices != nil {
		if len(value.codepointIndices) == 0 || startPos > len(value.codepointIndices) {
			return rawValue{data: ""}
		}
		var relevantIndices []int
		var lineData string
		if endPos >= len(value.codepointIndices) {
			relevantIndices = value.codepointIndices[startPos-1:]
			lineData = value.data[relevantIndices[0]:]
		} else {
			relevantIndices = value.codepointIndices[startPos-1 : endPos]
			lineData = value.data[relevantIndices[0]:value.codepointIndices[endPos]]
		}
		return rawValue{
			data:             trimFunc(lineData),
			codepointIndices: relevantIndices,
		}
	} else {
		if len(value.data) == 0 || startPos > len(value.data) {
			return rawValue{data: ""}
		}
		if endPos > len(value.data) {
			endPos = len(value.data)
		}
		return rawValue{
			data: trimFunc(value.data[startPos-1 : endPos]),
		}
	}
}

type valueSetter func(v reflect.Value, raw rawValue) error

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
		return structSetter(t)
	case reflect.String:
		return stringSetter
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return intSetter
	case reflect.Float32:
		return floatSetter(32)
	case reflect.Float64:
		return floatSetter(64)
	case reflect.Bool:
		return boolSetter
	}
	return unknownSetter
}

func structSetter(t reflect.Type) valueSetter {
	spec := cachedStructSpec(t)
	return func(v reflect.Value, raw rawValue) error {
		for i, fieldSpec := range spec.fieldSpecs {
			if !fieldSpec.ok {
				continue
			}
			rawValue := rawValueFromLine(raw, fieldSpec.startPos, fieldSpec.endPos, fieldSpec.format)
			err := fieldSpec.setter(v.Field(i), rawValue)
			if err != nil {
				sf := t.Field(i)
				return &UnmarshalTypeError{raw.data, sf.Type, t.Name(), sf.Name, err}
			}
		}
		return nil
	}
}

func unknownSetter(v reflect.Value, raw rawValue) error {
	return errors.New("fixedwidth: unknown type")
}

func nilSetter(v reflect.Value, _ rawValue) error {
	v.Set(reflect.Zero(v.Type()))
	return nil
}

func textUnmarshalerSetter(t reflect.Type, shouldAddr bool) valueSetter {
	return func(v reflect.Value, raw rawValue) error {
		if shouldAddr {
			v = v.Addr()
		}
		// set to zero value if this is nil
		if t.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(raw.data))
	}
}

func interfaceSetter(v reflect.Value, raw rawValue) error {
	return newValueSetter(v.Elem().Type())(v.Elem(), raw)
}

func ptrSetter(t reflect.Type) valueSetter {
	innerSetter := newValueSetter(t.Elem())
	return func(v reflect.Value, raw rawValue) error {
		if len(raw.data) <= 0 {
			return nilSetter(v, raw)
		}
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return innerSetter(reflect.Indirect(v), raw)
	}
}

func stringSetter(v reflect.Value, raw rawValue) error {
	v.SetString(raw.data)
	return nil
}

func intSetter(v reflect.Value, raw rawValue) error {
	if len(raw.data) < 1 {
		return nil
	}
	i, err := strconv.Atoi(raw.data)
	if err != nil {
		return err
	}
	v.SetInt(int64(i))
	return nil
}

func floatSetter(bitSize int) valueSetter {
	return func(v reflect.Value, raw rawValue) error {
		if len(raw.data) < 1 {
			return nil
		}
		f, err := strconv.ParseFloat(raw.data, bitSize)
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	}
}

func boolSetter(v reflect.Value, raw rawValue) error {
	if len(raw.data) == 0 {
		return nil
	}

	trimmedBool := strings.TrimSpace(raw.data)
	val, err := strconv.ParseBool(trimmedBool)
	if err != nil {
		return err
	}

	v.SetBool(val)
	return nil
}
