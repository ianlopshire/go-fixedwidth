// Package fixedwidth provides encoding and decoding for fixed-width formatted Data.
package fixedwidth

// Marshaler is the interface implemented by an object that can
// marshal itself into a fixed-width form.
//
// MarshalFixedWidth is provided a max width and should return
// the encoded value of the receiver. If the encoded value is
// longer than the max width, it will be truncated by the encoder.
// If the encoded value is shorter than the max width, it will be
// padded by the encoder.
type Marshaler interface {
	MarshalFixedWidth(width int) (data []byte, err error)
}

// Unmarshaler is the interface implemented by an object that can
// unmarshal a fixed-width representation of itself.
//
// The data passed to UnmarshalFixedWidth by the decoder will be
// the length of the field. No leading or trailing space will be
// removed.
//
// UnmarshalFixedWidth should be able to decode the form generated
// by MarshalFixedWidth.
type Unmarshaler interface {
	UnmarshalFixedWidth(data []byte) error
}
