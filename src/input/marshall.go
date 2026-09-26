package input

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/mazznoer/csscolorparser"
)

// CheckboxQueryToBool converts the "on" string we receive from chi from checkboxes to a bool.
// We're probably reinventing the wheel here ; refactor at will.
func CheckboxQueryToBool(queryParamValue []string) bool {
	out := false
	if len(queryParamValue) > 0 {
		if queryParamValue[0] != "off" && queryParamValue[0] != "0" {
			out = true
		}
	}
	return out
}

func QueryToString(queryParamValue []string, defaultValue string) string {
	out := defaultValue
	if len(queryParamValue) > 0 {
		out = queryParamValue[0]
	}
	return out
}

func QueryToColor(queryParamValue []string, defaultColor color.Color) color.Color {
	s := QueryToString(queryParamValue, "")
	c, err := csscolorparser.Parse(s)
	if err != nil {
		return defaultColor
	}
	return c
}

// DeserializeTally converts a comma-separated string of integer values to a slice of unsigned integers.
func DeserializeTally(tallyAsString string) ([]uint64, error) {
	spliceOfStrings := strings.Split(tallyAsString, ",")
	out := make([]uint64, len(spliceOfStrings))
	for i, s := range spliceOfStrings {
		t, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return out, err
		}
		out[i] = t
	}
	return out, nil
}
