package fixedwidth

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// parseTagWithFormat splits a struct fields fixed tag into its start position, end
// position, format, and padding character.
//
// If the tag is not valid, ok will be false.
func parseTag(tag string) (startPos, endPos int, format format, ok bool) {
	parts := strings.Split(tag, ",")
	if len(parts) < 2 || len(parts) > 4 {
		return 0, 0, defaultFormat, false
	}

	var err error
	if startPos, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, defaultFormat, false

	}
	if endPos, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, defaultFormat, false

	}
	if startPos > endPos || (startPos == 0 && endPos == 0) {
		return 0, 0, defaultFormat, false

	}

	format = defaultFormat

	if len(parts) >= 3 {
		alignment := alignment(parts[2])
		if alignment.Valid() {
			format.alignment = alignment
		}
	}

	if len(parts) >= 4 {
		v := parts[3]
		switch {
		case v == "_":
			format.padChar = ' '
		case parts[3] == "__":
			format.padChar = '_'
		case len(v) > 0:
			format.padChar = v[0]
		}
	}

	return startPos, endPos, format, true
}

type structSpec struct {
	// ll is the line length for the struct
	ll         int
	fieldSpecs []fieldSpec
}

type fieldSpec struct {
	startPos, endPos int
	encoder          valueEncoder
	codepointEncoder valueEncoder
	setter           valueSetter
	format           format
	ok               bool
}

func (s fieldSpec) len() int {
	return s.endPos - s.startPos + 1
}

func (s fieldSpec) getEncoder(useCodepointIndices bool) valueEncoder {
	if useCodepointIndices {
		return s.codepointEncoder
	}
	return s.encoder
}

func buildStructSpec(t reflect.Type) structSpec {
	ss := structSpec{
		fieldSpecs: make([]fieldSpec, t.NumField()),
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		startPos, endPos, format, ok := parseTag(f.Tag.Get("fixed"))
		if !ok {
			continue
		}

		ss.fieldSpecs[i].startPos = startPos
		ss.fieldSpecs[i].endPos = endPos
		ss.fieldSpecs[i].format = format
		ss.fieldSpecs[i].ok = ok

		if ss.fieldSpecs[i].endPos > ss.ll {
			ss.ll = ss.fieldSpecs[i].endPos
		}

		ss.fieldSpecs[i].encoder = newValueEncoder(f.Type, false)
		ss.fieldSpecs[i].codepointEncoder = newValueEncoder(f.Type, true)
		ss.fieldSpecs[i].setter = newValueSetter(f.Type)
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
