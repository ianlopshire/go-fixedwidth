package fixedwidth

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// parseTag splits a struct fields fixed tag into its start and end positions.
// If the tag is not valid, ok will be false.
func parseTag(tag string) (startPos, endPos int, ok bool) {
	parts := strings.Split(tag, ",")
	if len(parts) != 2 {
		return startPos, endPos, false
	}

	var err error
	if startPos, err = strconv.Atoi(parts[0]); err != nil {
		return startPos, endPos, false
	}
	if endPos, err = strconv.Atoi(parts[1]); err != nil {
		return startPos, endPos, false
	}
	if startPos > endPos || (startPos == 0 && endPos == 0) {
		return startPos, endPos, false
	}

	return startPos, endPos, true
}

type structSpec struct {
	// ll is the line length for the struct
	ll         int
	fieldSpecs []fieldSpec
}

type fieldSpec struct {
	name             string
	startPos, endPos int
	encoder          valueEncoder
	setter           valueSetter
	ok               bool
	isNumeric        bool
	format           *fieldFormat
}

type fieldFormat struct {
	rightAlign bool
	padChar    byte
}

func buildStructSpec(t reflect.Type) structSpec {
	ss := structSpec{
		fieldSpecs: make([]fieldSpec, t.NumField()),
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		ss.fieldSpecs[i].startPos, ss.fieldSpecs[i].endPos, ss.fieldSpecs[i].ok = parseTag(f.Tag.Get("fixed"))
		if ss.fieldSpecs[i].endPos > ss.ll {
			ss.ll = ss.fieldSpecs[i].endPos
		}
		ss.fieldSpecs[i].encoder = newValueEncoder(f.Type)
		ss.fieldSpecs[i].setter = newValueSetter(f.Type)
		ss.fieldSpecs[i].name = f.Name

		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
			ss.fieldSpecs[i].isNumeric = true
		}
	}
	return ss
}

var fieldSpecCache sync.Map // map[reflect.Type]structSpec

// cachedStructSpec is like buildStructSpec but cached to prevent duplicate work.
func cachedStructSpec(t reflect.Type) structSpec {
	if f, ok := fieldSpecCache.Load(t); ok {
		return f.(structSpec)
	}
	f, _ := fieldSpecCache.LoadOrStore(t, buildStructSpec(t))
	return f.(structSpec)
}
