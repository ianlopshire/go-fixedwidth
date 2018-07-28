package fixedwidth

import (
	"testing"
)

func TestParseTag(t *testing.T) {
	for _, tt := range []struct {
		name     string
		tag      string
		startPos int
		endPos   int
		leftpad  bool
		ok       bool
	}{
		{"Valid Tag", "0,10", 0, 10, false, true},
		{"Valid Tag Single position", "5,5", 5, 5, false, true},
		{"Tag Empty", "", 0, 0, false, false},
		{"Tag Too short", "0", 0, 0, false, false},
		{"Tag Too Long", "2,10,11", 0, 0, false, false},
		{"StartPos Not Integer", "hello,3", 0, 0, false, false},
		{"EndPos Not Integer", "3,hello", 0, 0, false, false},
		{"Tag Contains a Space", "4, 11", 0, 0, false, false},
		{"Tag Interval Invalid", "14,5", 0, 0, false, false},
		{"Tag Both Positions Zero", "0,0", 0, 0, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseTag(tt.tag)
			if tt.ok != (err == nil) {
				t.Errorf("parseTag() shouldError %v, have %v", !tt.ok, err)
			}

			// only check startPos and endPos if valid tags are expected
			if tt.ok {
				if tt.startPos != spec.startPos {
					t.Errorf("parseTag() startPos want %v, have %v", tt.startPos, spec.startPos)
				}

				if tt.endPos != spec.endPos {
					t.Errorf("parseTag() endPos want %v, have %v", tt.endPos, spec.endPos)
				}

				if tt.leftpad != spec.leftpad {
					t.Errorf("parseTag() lefpad expected %v, have %v", tt.leftpad, spec.leftpad)
				}
			}
		})
	}
}
