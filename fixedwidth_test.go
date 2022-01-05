package fixedwidth

var (
	nilFloat64 *float64
	nilFloat32 *float32
	nilInt     *int
	nilString  *string
	nilUint    *uint
)

func float64p(v float64) *float64 { return &v }
func float32p(v float32) *float32 { return &v }
func intp(v int) *int             { return &v }
func int64p(v int64) *int64       { return &v }
func int32p(v int32) *int32       { return &v }
func int16p(v int16) *int16       { return &v }
func int8p(v int8) *int8          { return &v }
func stringp(v string) *string    { return &v }
func uintp(v uint) *uint          { return &v }
func boolp(v bool) *bool          { return &v }

// EncodableString is a string that implements the encoding TextUnmarshaler and TextMarshaler interface.
// This is useful for testing.
type EncodableString struct {
	S   string
	Err error
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *EncodableString) UnmarshalText(text []byte) error {
	s.S = string(text)
	return s.Err
}

// MarshalText implements encoding.TextUnmarshaler.
func (s EncodableString) MarshalText() ([]byte, error) {
	return []byte(s.S), s.Err
}
