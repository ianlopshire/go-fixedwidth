package fixedwidth

import (
	"bytes"
	"errors"
	"unicode/utf8"
)

// lineBuilder is a multibyte character aware buffer that can be used to efficiently build
// a line of fixed width text.
type lineBuilder struct {
	data []byte

	// Used when `SetUseCodepointIndices` has been called on `Encoder`. A
	// mapping of codepoint indices into the bytes. So the `codepointIndices[n]` is the
	// starting position for the n-th codepoint in `bytes`.
	codepointIndices []int
}

// newLineBuilder makes a new lineBuilder. The line is filled with the provided fillChar.
func newLineBuilder(len, cap int, fillChar byte) *lineBuilder {
	data := make([]byte, len, cap)

	// Fill the buffer with the fill character.
	data[0] = fillChar
	filled := 1
	for filled < len {
		copy(data[filled:], data[:filled])
		filled *= 2
	}

	buff := &lineBuilder{
		data: data,
	}

	return buff
}

// lineBufferFromValue creates a lineBuilder from a rawValue.
func lineBufferFromValue(value rawValue) *lineBuilder {
	buff := newLineBuilder(value.len(), value.byteLen(), ' ')
	buff.WriteValue(0, value)
	return buff
}

// WriteValue writes the given value to the lineBuilder at the give start index.
func (b *lineBuilder) WriteValue(start int, value rawValue) {
	// Fast path for ascii only operation.
	if !b.hasMultiByteChar() && !value.hasMultiByteChar() {
		copy(b.data[start:], value.data)
		return
	}

	// If this is the first time a multibyte character has been encountered, the codepoint
	// indices need to be initialized.
	if !b.hasMultiByteChar() && value.hasMultiByteChar() {
		b.initializeIndices()
	}

	end := start + value.len() - 1

	// Calculate the byte start and end indices accounting for any multibyte characters.
	byteStart := b.codepointIndices[start]
	byteEnd := b.byteEndIndex(end)

	writeSpan := b.data[byteStart : byteEnd+1]

	// Ensure the there is space for the value being written. adjustByteSpan will grow or
	// shrink the byte span if required.
	byteDiff := value.byteLen() - len(writeSpan)
	if byteDiff != 0 {
		b.adjustByteSpan(end, byteDiff)

		// Correct the writeSpan after the adjustment.
		byteEnd = b.byteEndIndex(end)
		writeSpan = b.data[byteStart : byteEnd+1]
	}

	// Write the value to the buffer
	copy(b.data[byteStart:byteEnd+1], value.data)

	// Correct the indices for the value that was just written. This only needs to happen
	// if we adjusted the write-span or the new value contains multibyte characters.
	if byteDiff != 0 || value.hasMultiByteChar() {
		b.correctIndices(start, value)
	}
}

// WriteASCII writes an ascii string to the line builder.
func (b *lineBuilder) WriteASCII(start int, data string) {
	v, _ := newRawValue(data, false)
	b.WriteValue(start, v)
}

func (b *lineBuilder) String() string {
	return string(b.data)
}

func (b *lineBuilder) AsRawValue() rawValue {
	return rawValue{
		data:             b.String(),
		codepointIndices: b.codepointIndices,
	}
}

func (b *lineBuilder) initializeIndices() {
	b.codepointIndices = make([]int, len(b.data))
	for i := range b.codepointIndices {
		b.codepointIndices[i] = i
	}
}

func (b *lineBuilder) correctIndices(start int, value rawValue) {
	firstIndex := b.byteEndIndex(start-1) + 1

	// Fast path for ascii values – there is no need to individually calculate the
	// indices.
	if !value.hasMultiByteChar() {
		for i := 0; i < value.len(); i++ {
			b.codepointIndices[start+i] = firstIndex + i
		}
		return
	}

	for i, s := range value.codepointIndices {
		b.codepointIndices[start+i] = firstIndex + s
	}
}

func (b *lineBuilder) adjustByteSpan(end, diff int) {
	byteEnd := b.byteEndIndex(end)

	switch {
	case diff < 0:
		// shorten buffer data
		copy(b.data[byteEnd+diff:], b.data[byteEnd:])
		b.data = b.data[:len(b.data)+diff]

	case diff > 0:
		// expand buffer data
		b.data = append(b.data, bytes.Repeat([]byte{' '}, diff)...)
		copy(b.data[byteEnd+diff:], b.data[byteEnd:])

	}

	// correct indices
	for i := end + 1; i < len(b.codepointIndices); i++ {
		b.codepointIndices[i] += diff
	}
}

func (b *lineBuilder) byteStartIndex(start int) int {
	if b.codepointIndices == nil {
		return start
	}
	return b.codepointIndices[start]
}

func (b *lineBuilder) byteEndIndex(end int) int {
	if b.codepointIndices == nil {
		return end
	}
	if end == len(b.codepointIndices)-1 {
		return len(b.data) - 1
	}
	return b.codepointIndices[end+1] - 1
}

func (b *lineBuilder) hasMultiByteChar() bool {
	return b.codepointIndices != nil
}

type rawValue struct {
	data string
	// Used when `SetUseCodepointIndices` has been called on `Decoder` or `Encoder`. A
	// mapping of codepoint indices into the bytes. So the `codepointIndices[n]` is the
	// starting position for the n-th codepoint in `bytes`.
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

func (v rawValue) len() int {
	if v.codepointIndices == nil {
		return len(v.data)
	}
	return len(v.codepointIndices)
}

func (v rawValue) byteLen() int {
	return len(v.data)
}

func (v rawValue) hasMultiByteChar() bool {
	return v.codepointIndices != nil
}

func (v rawValue) byteStartIndex(start int) int {
	if v.codepointIndices == nil {
		return start
	}
	return v.codepointIndices[start]
}

func (v rawValue) byteEndIndex(end int) int {
	if v.codepointIndices == nil {
		return end
	}
	if end == len(v.codepointIndices)-1 {
		return len(v.data) - 1
	}
	return v.codepointIndices[end+1] - 1
}

func (v rawValue) slice(start, end int) (rawValue, error) {
	d := v.data[v.byteStartIndex(start) : v.byteEndIndex(end)+1]
	return newRawValue(d, v.hasMultiByteChar())
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
