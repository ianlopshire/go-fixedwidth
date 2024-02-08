package fixedwidth

const (
	defaultAlignment alignment = "default"
	noAlignment      alignment = "none"
	right            alignment = "right"
	left             alignment = "left"
)

const (
	defaultPadChar = ' '
)

var defaultFormat = format{
	alignment: defaultAlignment,
	padChar:   defaultPadChar,
}

type format struct {
	alignment alignment
	padChar   byte
}

type alignment string

func (a alignment) Valid() bool {
	switch a {
	case defaultAlignment, right, left, noAlignment:
		return true
	default:
		return false
	}
}
