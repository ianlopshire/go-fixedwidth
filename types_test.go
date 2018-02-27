package fixedwidth

import (
	"bytes"
	"testing"
)

func TestFloat_MarshalFixedWidth(t *testing.T) {
	for _, tt := range []struct {
		name      string
		f         Float
		width     int
		data      []byte
		shouldErr bool
	}{
		{
			name:      "zero",
			f:         0,
			width:     10,
			data:      []byte(`0.00000000`),
			shouldErr: false,
		},
		{
			name:      "whole number",
			f:         11,
			width:     10,
			data:      []byte(`11.0000000`),
			shouldErr: false,
		},
		{
			name:      "negative whole number",
			f:         -11,
			width:     10,
			data:      []byte(`-11.000000`),
			shouldErr: false,
		},
		{
			name:      "rational number",
			f:         11.234,
			width:     10,
			data:      []byte(`11.2340000`),
			shouldErr: false,
		},
		{
			name:      "negative rational number",
			f:         -11.234,
			width:     10,
			data:      []byte(`-11.234000`),
			shouldErr: false,
		},
		{
			name:      "zero precision",
			f:         1234567891.234,
			width:     10,
			data:      []byte(`1234567891`),
			shouldErr: false,
		},
		{
			name:      "negative zero precision",
			f:         -123456789.234,
			width:     10,
			data:      []byte(`-123456789`),
			shouldErr: false,
		},
		{
			name:      "error too long",
			f:         12345678912.234,
			width:     10,
			shouldErr: true,
		},
		{
			name:      "error negative too long",
			f:         -1234567891.234,
			width:     10,
			shouldErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.f.MarshalFixedWidth(tt.width)
			if err != nil != tt.shouldErr {
				t.Errorf("MarshalFixedWidth() err have %v, want %v (%v)", err != nil, tt.shouldErr, err)
			}
			if !bytes.Equal(data, tt.data) {
				t.Errorf("MarshalFixedWidth() data have %s, want %s", data, tt.data)
			}
		})
	}
}
