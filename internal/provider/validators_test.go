package provider

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestValidateIntegerInRange(t *testing.T) {
	validIntegers := []int{-259, 0, 1, 5, 999}
	min := -259
	max := 999
	for _, v := range validIntegers {
		if diags := validateIntegerInRange(min, max)(v, *new(cty.Path)); len(diags) != 0 {
			t.Fatalf("%q should be an integer in range (%d, %d): %q", v, min, max, diags)
		}
	}

	invalidIntegers := []int{-260, -99999, 1000, 25678}
	for _, v := range invalidIntegers {
		if diags := validateIntegerInRange(min, max)(v, *new(cty.Path)); len(diags) == 0 {
			t.Fatalf("%q should be an integer in range (%d, %d)", v, min, max)
		}
	}
}

func TestValidateIntegerGeqThan0(t *testing.T) {
	v := 1
	if _, error := validateIntegerGeqThan(0)(v, "name"); error != nil {
		t.Fatalf("%q should be an integer greater than 0", v)
	}

	v = -4
	if _, error := validateIntegerGeqThan(0)(v, "name"); error == nil {
		t.Fatalf("%q should be an invalid integer smaller than 0", v)
	}
}

func TestValidateFloatRatio(t *testing.T) {
	v := 0.9
	if _, error := validateFloatRatio()(v, "name"); error != nil {
		t.Fatalf("%v should be a float between 0.0 and 1.0", v)
	}

	v = -4.5
	if _, error := validateFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float smaller than 0.0", v)
	}

	v = 1.1
	if _, error := validateFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float greater than 1.0", v)
	}
}

func TestValidateStringIsFloatRatio(t *testing.T) {
	v := "0.9"
	if _, error := validateStringIsFloatRatio()(v, "name"); error != nil {
		t.Fatalf("%v should be a float between 0.0 and 1.0", v)
	}

	v = "-4.5"
	if _, error := validateStringIsFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float smaller than 0.0", v)
	}

	v = "1.1"
	if _, error := validateStringIsFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float greater than 1.0", v)
	}
	v = "false"
	if _, error := validateStringIsFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float because it is a bool in a string", v)
	}
	w := false
	if _, error := validateStringIsFloatRatio()(w, "name"); error == nil {
		t.Fatalf("%v should be an invalid float because it is a bool", v)
	}
	i := 0
	if _, error := validateStringIsFloatRatio()(i, "name"); error != nil {
		t.Fatalf("%v should be a valid float because int can be casted", v)
	}
	i = 1
	if _, error := validateStringIsFloatRatio()(i, "name"); error != nil {
		t.Fatalf("%v should be a valid float because int can be casted", v)
	}
	i = 4
	if _, error := validateStringIsFloatRatio()(i, "name"); error == nil {
		t.Fatalf("%v should be an invalid float because it is an int out of range", v)
	}
}

func TestValidateDurationGeq0(t *testing.T) {
	v := "1ms"
	if _, error := validateDurationGeq0()(v, "name"); error != nil {
		t.Fatalf("%v should be a valid durarion", v)
	}

	v = "-2h"
	if _, error := validateDurationGeq0()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid duration smaller than 0", v)
	}
}

func TestValidateStringMatchesPattern(t *testing.T) {
	pattern := `^(pause|continue-mate|break)$`
	v := "pause"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error != nil {
		t.Fatalf("%q should match the pattern", v)
	}
	v = "doesnotmatch"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error == nil {
		t.Fatalf("%q should not match the pattern", v)
	}
	v = "continue-mate"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error != nil {
		t.Fatalf("%q should match the pattern", v)
	}
}

func TestValidateStringShouldBeBase64EncodedDiag(t *testing.T) {
	v := `YmtzbGRrc2xka3NkMjM4MQ==`
	if diags := validateStringIsBase64Encoded(v, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%q should be base64 decodeable", v)
	}

	v = `%&df#3NkMjM4MQ==`
	if diags := validateStringIsBase64Encoded(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%q should NOT be base64 decodeable", v)
	}
}
