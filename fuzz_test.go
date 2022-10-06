//go:build go1.18
// +build go1.18

package fixedwidth_test

import (
	"bytes"
	"testing"

	"github.com/ianlopshire/go-fixedwidth"
)

func FuzzUnmarshal(f *testing.F) {
	unmarshal := func(data []byte, v interface{}, useCodepointIndices bool) error {
		if useCodepointIndices {
			dec := fixedwidth.NewDecoder(bytes.NewReader(data))
			dec.SetUseCodepointIndices(useCodepointIndices)
			return dec.Decode(v)
		}
		return fixedwidth.Unmarshal(data, v)
	}

	marshal := func(v interface{}, useCodepointIndices bool) ([]byte, error) {
		buff := bytes.NewBuffer(nil)
		enc := fixedwidth.NewEncoder(buff)
		enc.SetUseCodepointIndices(useCodepointIndices)

		if err := enc.Encode(v); err != nil {
			return nil, err
		}

		return buff.Bytes(), nil
	}

	typs := []func() interface{}{
		func() interface{} {
			return new([]struct {
				F string `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F string `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F int `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F int64 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F int32 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F int16 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F int8 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F uint `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F uint64 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F uint32 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F uint16 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F uint8 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F float32 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F float64 `fixed:"1,10"`
			})
		},
		func() interface{} {
			return new(struct {
				F bool `fixed:"1,10"`
			})
		},
	}

	f.Add([]byte(`\n`))
	f.Add([]byte(`foo       `))
	f.Add([]byte(`foo       ` + "\n" + `foo       `))
	f.Add([]byte(`føø       `))
	f.Add([]byte(`true      `))
	f.Add([]byte(`123       `))
	f.Add([]byte(`123.456   `))
	f.Add([]byte(`-123      `))

	f.Fuzz(func(t *testing.T, b []byte) {
		for _, typ := range typs {
			for _, useCodepointIndices := range []bool{true, false} {
				i := typ()
				if err := unmarshal(b, i, useCodepointIndices); err != nil {
					continue
				}

				encoded, err := marshal(i, useCodepointIndices)
				if err != nil {
					t.Fatalf("failed to marshal: %s", err)
				}
				if err := unmarshal(encoded, i, useCodepointIndices); err != nil {
					t.Fatalf("failed to roundtrip: %s", err)
				}
			}
		}
	})
}
