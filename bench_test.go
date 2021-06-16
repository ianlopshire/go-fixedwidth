package fixedwidth

import (
	"bytes"
	"testing"
)

type mixedData struct {
	F1  string   `fixed:"1,10"`
	F2  *string  `fixed:"11,20"`
	F3  int64    `fixed:"21,30"`
	F4  *int64   `fixed:"31,40"`
	F5  int32    `fixed:"41,50"`
	F6  *int32   `fixed:"51,60"`
	F7  int16    `fixed:"61,70"`
	F8  *int16   `fixed:"71,80"`
	F9  int8     `fixed:"81,90"`
	F10 *int8    `fixed:"91,100"`
	F11 float64  `fixed:"101,110"`
	F12 *float64 `fixed:"111,120"`
	F13 float32  `fixed:"121,130"`
	F14 bool     `fixed:"131,140"`
	F15 bool     `fixed:"141,150"`
	//F14 *float32 `fixed:"131,140"`
}

var mixedDataInstance = mixedData{"foo", stringp("foo"), 42, int64p(42), 42, int32p(42), 42, int16p(42), 42, int8p(42), 4.2, float64p(4.2), 4.2, false, true} //,float32p(4.2)}

func BenchmarkUnmarshal_MixedData_1(b *testing.B) {
	data := []byte(`       foo       foo        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2     false         t`)
	var v mixedData
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkUnmarshal_MixedData_1000(b *testing.B) {
	data := bytes.Repeat([]byte(`       foo       foo        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2     false         t`+"\n"), 100)
	var v []mixedData
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkUnmarshal_MixedData_100000(b *testing.B) {
	data := bytes.Repeat([]byte(`       foo       foo        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2     false         t`+"\n"), 10000)
	var v []mixedData
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkDecode_CodePoints_MixedData_1_Ascii(b *testing.B) {
	data := []byte(`       foo       foo        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2     false         t`)
	var v mixedData
	for i := 0; i < b.N; i++ {
		d := NewDecoder(bytes.NewReader(data))
		d.SetUseCodepointIndices(true)
		_ = d.Decode(&v)
	}
}

func BenchmarkDecode_CodePoints_MixedData_1_UTF8(b *testing.B) {
	data := []byte(`       f☃☃       f☃☃        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2     false         t`)
	var v mixedData
	for i := 0; i < b.N; i++ {
		d := NewDecoder(bytes.NewReader(data))
		d.SetUseCodepointIndices(true)
		_ = d.Decode(&v)
	}
}

func BenchmarkUnmarshal_String(b *testing.B) {
	data := []byte(`foo       `)
	var v struct {
		F1 string `fixed:"1,10"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkUnmarshal_StringPtr(b *testing.B) {
	data := []byte(`foo       `)
	var v struct {
		F1 *string `fixed:"1,10"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkUnmarshal_Int64(b *testing.B) {
	data := []byte(`42       `)
	var v struct {
		F1 int64 `fixed:"1,10"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkUnmarshal_Float64(b *testing.B) {
	data := []byte(`4.2      `)
	var v struct {
		F1 float64 `fixed:"1,10"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(data, &v)
	}
}

func BenchmarkMarshal_MixedData_1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Marshal(mixedDataInstance)
	}
}

func BenchmarkMarshal_MixedData_1000(b *testing.B) {
	v := make([]mixedData, 1000)
	for i := range v {
		v[i] = mixedDataInstance
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_MixedData_100000(b *testing.B) {
	v := make([]mixedData, 100000)
	for i := range v {
		v[i] = mixedDataInstance
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_String(b *testing.B) {
	v := struct {
		F1 string `fixed:"1,10"`
	}{
		F1: "foo",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_StringPtr(b *testing.B) {
	v := struct {
		F1 *string `fixed:"1,10"`
	}{
		F1: stringp("foo"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_Int64(b *testing.B) {
	v := struct {
		F1 int64 `fixed:"1,10"`
	}{
		F1: 42,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_Float64(b *testing.B) {
	v := struct {
		F1 float64 `fixed:"1,10"`
	}{
		F1: 4.2,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}

func BenchmarkMarshal_Bool(b *testing.B) {
	v := struct {
		F1 bool `fixed:"1,10"`
	}{
		F1: false,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal(v)
	}
}
