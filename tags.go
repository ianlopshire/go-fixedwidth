package fixedwidth

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var errEmptyTag = errors.New("empty tag")

// parseTag returns the fieldSpec from the tag params.
// If the tag is not valid, it will return an error.
func parseTag(tag string) (fieldSpec, error) {
	if tag == "" {
		return fieldSpec{}, errEmptyTag
	}
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return fieldSpec{}, errors.Errorf("missing start and end positions: %q", tag)
	}
	if len(parts) > 3 {
		return fieldSpec{}, errors.Errorf("invalid fixed tag: %q", tag)
	}

	var (
		spec fieldSpec
		err  error
	)
	if spec.startPos, err = strconv.Atoi(parts[0]); err != nil {
		return fieldSpec{}, err
	}
	if spec.endPos, err = strconv.Atoi(parts[1]); err != nil {
		return fieldSpec{}, err
	}
	if spec.startPos > spec.endPos || (spec.startPos == 0 && spec.endPos == 0) {
		return fieldSpec{}, errors.Errorf("end position (%d) ahead of start position (%d)", spec.startPos, spec.endPos)
	}
	if len(parts) == 2 {
		return spec, nil
	}

	if parts[2] != "leftpad" {
		return fieldSpec{}, errors.Errorf("unknown fied tag option %q", parts[2])
	}
	spec.leftpad = true

	return spec, nil
}
