package fixedwidth

import (
	"bufio"
	"bytes"
	"encoding"
	"io"
	"reflect"
	"strconv"
	"unicode"
)

// Unmarshal parses the fixed width encoded data and stores the
// result in the value pointed to by v. If v is nil or not a
// pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	d := Decoder{
		data: bufio.NewReader(bytes.NewReader(data)),
	}
	return d.Decode(v)
}

// A Decoder reads and decodes fixed width data from an input stream.
type Decoder struct {
	data *bufio.Reader
	done bool
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

// An UnmarshalTypeError describes a  value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // the raw value
	Type   reflect.Type // type of Go value it could not be assigned to
	Struct string       // name of the struct type containing the field
	Field  string       // name of the field holding the Go value
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "fixedwidth: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	}
	return "fixedwidth: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}

// Decode reads from its input and stores the decoded data the value
// pointed to by v.
//
// In the case that v points to a struct value, Decode will read a
// single line from the input.
//
// In the case that v points to a slice value, Decode will read until
// the end of its input.
func (d *Decoder) Decode(v interface{}) (error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	switch reflect.Indirect(reflect.ValueOf(v)).Kind() {
	case reflect.Slice:
		return d.array(reflect.ValueOf(v).Elem())
	case reflect.Struct:
		return d.object(reflect.ValueOf(v).Elem())
	default:
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
}

func (d *Decoder) array(v reflect.Value) (err error) {
	ct := v.Type().Elem()
	for {
		nv := reflect.New(ct).Elem()
		err := d.object(nv)
		if err != nil {
			return err
		}
		if d.done {
			break
		}
		v.Set(reflect.Append(v, nv))
	}
	return nil
}

func (d *Decoder) object(v reflect.Value) (err error) {
	// TODO: properly handle prefixed lines
	line, _, err := d.data.ReadLine()
	if err == io.EOF {
		d.done = true
		return nil
	} else if err != nil {
		return err
	}

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
		rawValue := getRawValue(line, startPos, endPos)

		if tu, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			err := tu.UnmarshalText(rawValue)
			if err != nil {
				return err
			}
			continue
		}

		valid := false
		switch sf.Type.Kind() {
		case reflect.String:
			valid = setString(fv, rawValue)
		case reflect.Int:
			valid = setInt(fv, rawValue)
		case reflect.Float64:
			valid = setFloat(fv, rawValue)
		}
		if !valid {
			return &UnmarshalTypeError{string(rawValue), sf.Type, t.Name(), sf.Name}
		}
	}
	return nil
}

func getRawValue(line []byte, startPos, endPos int) []byte {
	if len(line) == 0 || startPos >= len(line) {
		return []byte{}
	}
	if endPos > len(line) {
		endPos = len(line)
	}
	return bytes.TrimRightFunc(line[startPos-1:endPos], unicode.IsSpace)
}

func setString(fv reflect.Value, rawValue []byte) bool {
	fv.SetString(string(rawValue))
	return true
}

func setInt(fv reflect.Value, rawValue []byte) bool {
	if len(rawValue) < 1 {
		return true
	}
	i, err := strconv.Atoi(string(rawValue))
	if err != nil {
		return false
	}
	fv.SetInt(int64(i))
	return true
}

func setFloat(fv reflect.Value, rawValue []byte) bool {
	if len(rawValue) < 1 {
		return true
	}
	f, err := strconv.ParseFloat(string(rawValue), 64)
	if err != nil {
		return false
	}
	fv.SetFloat(f)
	return true
}
