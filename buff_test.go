package fixedwidth

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMakeLineBuffer(t *testing.T) {
	for _, tt := range []struct {
		name     string
		len      int
		cap      int
		fillChar byte

		expectData []byte
	}{
		{
			name:       "base case",
			len:        5,
			cap:        10,
			fillChar:   ' ',
			expectData: []byte(`     `),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buff := newLineBuilder(tt.len, tt.cap, tt.fillChar)

			if len(buff.data) != tt.len {
				t.Errorf("newLineBuilder() expected len %v, have %v", tt.len, len(buff.data))
			}
			if cap(buff.data) != tt.cap {
				t.Errorf("newLineBuilder() expected cap %v, have %v", tt.cap, cap(buff.data))
			}
			if !bytes.Equal(buff.data, tt.expectData) {
				t.Errorf("newLineBuilder() expected data %q, have %q", string(tt.expectData), string(buff.data))
			}
		})
	}
}

func TestLineBuffer_writeValue(t *testing.T) {
	for _, tt := range []struct {
		name string

		buff *lineBuilder

		start int
		value string

		expectData    []byte
		expectIndices []int
	}{
		{
			name:          "ascii to empty buff (fill)",
			buff:          newLineBuilder(3, 3, ' '),
			start:         0,
			value:         "foo",
			expectData:    []byte(`foo`),
			expectIndices: nil,
		},
		{
			name:          "ascii to empty buff (start)",
			buff:          newLineBuilder(5, 5, ' '),
			start:         0,
			value:         "foo",
			expectData:    []byte(`foo  `),
			expectIndices: nil,
		},
		{
			name:          "ascii to empty buff (end)",
			buff:          newLineBuilder(5, 5, ' '),
			start:         2,
			value:         "foo",
			expectData:    []byte(`  foo`),
			expectIndices: nil,
		},
		{
			name:          "ascii to empty buff (mid)",
			buff:          newLineBuilder(5, 5, ' '),
			start:         1,
			value:         "foo",
			expectData:    []byte(` foo `),
			expectIndices: nil,
		},
		{
			name:          "multibyte to empty buff (fill)",
			buff:          newLineBuilder(3, 5, ' '),
			start:         0,
			value:         "føø",
			expectData:    []byte(`føø`),
			expectIndices: []int{0, 1, 3},
		},
		{
			name:          "multibyte to empty buff (fill)(past cap)",
			buff:          newLineBuilder(3, 3, ' '),
			start:         0,
			value:         "føø",
			expectData:    []byte(`føø`),
			expectIndices: []int{0, 1, 3},
		},
		{
			name:          "multibyte to empty buff (start)",
			buff:          newLineBuilder(5, 10, ' '),
			start:         0,
			value:         "føø",
			expectData:    []byte(`føø  `),
			expectIndices: []int{0, 1, 3, 5, 6},
		},
		{
			name:          "multibyte to empty buff (end)",
			buff:          newLineBuilder(5, 10, ' '),
			start:         2,
			value:         "føø",
			expectData:    []byte(`  føø`),
			expectIndices: []int{0, 1, 2, 3, 5},
		},
		{
			name:          "multibyte to empty buff (mid)",
			buff:          newLineBuilder(5, 10, ' '),
			start:         1,
			value:         "føø",
			expectData:    []byte(` føø `),
			expectIndices: []int{0, 1, 2, 4, 6},
		},
		{
			name:          "multibyte to multibyte (fill)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         0,
			value:         "øøø",
			expectData:    []byte(`øøø`),
			expectIndices: []int{0, 2, 4},
		},
		{
			name:          "multibyte to multibyte (mid)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         1,
			value:         "ø",
			expectData:    []byte(`åøå`),
			expectIndices: []int{0, 2, 4},
		},
		{
			name:          "multibyte to multibyte (start)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         0,
			value:         "ø",
			expectData:    []byte(`øåå`),
			expectIndices: []int{0, 2, 4},
		},
		{
			name:          "multibyte to multibyte (end)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         2,
			value:         "ø",
			expectData:    []byte(`ååø`),
			expectIndices: []int{0, 2, 4},
		},
		{
			name:          "mixed to multibyte (fill)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         0,
			value:         "føø",
			expectData:    []byte(`føø`),
			expectIndices: []int{0, 1, 3},
		},
		{
			name:          "mixed to multibyte (fill)",
			buff:          lineBufferFromValue(mustRawValue("ååå")),
			start:         0,
			value:         "øøf",
			expectData:    []byte(`øøf`),
			expectIndices: []int{0, 2, 4},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.buff.WriteValue(tt.start, mustRawValue(tt.value))
			if !bytes.Equal(tt.buff.data, tt.expectData) {
				t.Errorf("WriteValue() expected data %q, have %q", string(tt.expectData), string(tt.buff.data))
				t.Errorf("WriteValue() expected data %v, have %v", tt.expectData, tt.buff.data)
			}
			if !reflect.DeepEqual(tt.buff.codepointIndices, tt.expectIndices) {
				t.Errorf("WriteValue() expected indices %v, have %v", tt.expectIndices, tt.buff.codepointIndices)
			}
		})
	}

}

func TestLineBuffer_byteEndIndex(t *testing.T) {
	foo := &lineBuilder{
		data:             []byte(`foo`),
		codepointIndices: []int{0, 1, 2},
	}
	føø := &lineBuilder{
		data:             []byte(`føø`),
		codepointIndices: []int{0, 1, 3},
	}

	for _, tt := range []struct {
		name        string
		buff        *lineBuilder
		end         int
		expectIndex int
	}{
		{"foo[0]", foo, 0, 0},
		{"foo[1]", foo, 1, 1},
		{"foo[2]", foo, 2, 2},
		{"føø[0]", føø, 0, 0},
		{"føø[1]", føø, 1, 2},
		{"føø[2]", føø, 2, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			index := tt.buff.byteEndIndex(tt.end)
			if index != tt.expectIndex {
				t.Errorf("byteEndIndex() expected %v, have %v", tt.expectIndex, index)
			}
		})
	}
}

func TestLineBuffer_byteStartIndex(t *testing.T) {
	foo := &lineBuilder{
		data:             []byte(`foo`),
		codepointIndices: []int{0, 1, 2},
	}
	føø := &lineBuilder{
		data:             []byte(`føø`),
		codepointIndices: []int{0, 1, 3},
	}

	for _, tt := range []struct {
		name        string
		buff        *lineBuilder
		start       int
		expectIndex int
	}{
		{"foo[0]", foo, 0, 0},
		{"foo[1]", foo, 1, 1},
		{"foo[2]", foo, 2, 2},
		{"føø[0]", føø, 0, 0},
		{"føø[1]", føø, 1, 1},
		{"føø[2]", føø, 2, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			index := tt.buff.byteStartIndex(tt.start)
			if index != tt.expectIndex {
				t.Errorf("byteStartIndex() expected %v, have %v", tt.expectIndex, index)
			}
		})
	}
}

func TestLineBuffer_adjustByteSpan(t *testing.T) {
	for _, tt := range []struct {
		name string

		buff *lineBuilder
		end  int
		diff int

		expectData    []byte
		expectIndices []int
	}{
		{
			name: "shorten byte span (end)",
			buff: &lineBuilder{
				data:             []byte(`føø`),
				codepointIndices: []int{0, 1, 3},
			},
			end:           2,
			diff:          -1,
			expectData:    append([]byte(`fø`), '\xb8'),
			expectIndices: []int{0, 1, 3},
		},
		{
			name: "shorten byte span (mid)",
			buff: &lineBuilder{
				data:             []byte(`føø`),
				codepointIndices: []int{0, 1, 3},
			},
			end:           1,
			diff:          -1,
			expectData:    append([]byte{'f', '\xb8'}, "ø"...),
			expectIndices: []int{0, 1, 2},
		},
		{
			name: "expand byte span (end)",
			buff: &lineBuilder{
				data:             []byte(`føø`),
				codepointIndices: []int{0, 1, 3},
			},
			end:           2,
			diff:          1,
			expectData:    append([]byte(`føø`), '\xb8'),
			expectIndices: []int{0, 1, 3},
		},

		{
			name: "expand byte span (mid)",
			buff: &lineBuilder{
				data:             []byte(`foø`),
				codepointIndices: []int{0, 1, 2},
			},
			end:           1,
			diff:          1,
			expectData:    []byte("fooø"),
			expectIndices: []int{0, 1, 3},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.buff.adjustByteSpan(tt.end, tt.diff)
			if !bytes.Equal(tt.buff.data, tt.expectData) {
				t.Errorf("adjustByteSpan() expected date %q, have %q", string(tt.expectData), string(tt.buff.data))
				t.Errorf("adjustByteSpan() expected date %v, have %v", tt.expectData, tt.buff.data)
			}
			if !reflect.DeepEqual(tt.buff.codepointIndices, tt.expectIndices) {
				t.Errorf("adjustByteSpan() expected indices %v, have %v", tt.expectIndices, tt.buff.codepointIndices)
			}
		})
	}
}

func TestRawValue_len(t *testing.T) {
	for _, tt := range []struct {
		value rawValue
		want  int
	}{
		{mustRawValue(""), 0},
		{mustRawValue("foo"), 3},
		{mustRawValue("føø"), 3},
	} {
		t.Run(tt.value.data, func(t *testing.T) {
			if l := tt.value.len(); l != tt.want {
				t.Errorf("len() expected %v, have %v", tt.want, l)

			}
		})
	}
}

func mustRawValue(data string) rawValue {
	v, err := newRawValue(data, true)
	if err != nil {
		panic(err)
	}
	return v
}
