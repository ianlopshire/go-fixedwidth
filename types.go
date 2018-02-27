package fixedwidth

import (
	"errors"
	"math"
	"strconv"
)

type Float float64

func (f Float) MarshalFixedWidth(width int) (data []byte, err error) {
	var l, p int

	if f > 0 {
		l = int(math.Log10(float64(f))) + 2
	} else if f < 0 {
		l = int(math.Log10(math.Abs(float64(f)))) + 3
	} else {
		l = 2
	}

	if l-1 > width {
		return nil, errors.New("formatted float with 0 precision longer than field width")
	}

	p = width - l
	if p < 0 {
		p = 0
	}

	s := strconv.FormatFloat(float64(f), 'f', p, 64)
	return []byte(s), nil
}
