package provider

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestValidateIntegerGeqThan0(t *testing.T) {
	v := 1
	if diags := validateIntegerGeqThan(0)(v, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%d should be an integer greater than 0", v)
	}

	v = -4
	if diags := validateIntegerGeqThan(0)(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%d should be an invalid integer smaller than 0", v)
	}
}

func TestValidateStringIsFloatRatio(t *testing.T) {
	v := "0.9"
	if diags := validateStringIsFloatRatio()(v, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%v should be a float between 0.0 and 1.0", v)
	}

	v = "-4.5"
	if diags := validateStringIsFloatRatio()(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%v should be an invalid float smaller than 0.0", v)
	}

	v = "1.1"
	if diags := validateStringIsFloatRatio()(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%v should be an invalid float greater than 1.0", v)
	}
	v = "false"
	if diags := validateStringIsFloatRatio()(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%v should be an invalid float because it is a bool in a string", v)
	}
	w := false
	if diags := validateStringIsFloatRatio()(w, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%v should be an invalid float because it is a bool", v)
	}
	i := 0
	if diags := validateStringIsFloatRatio()(i, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%v should be a valid float because int can be casted", v)
	}
	i = 1
	if diags := validateStringIsFloatRatio()(i, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%v should be a valid float because int can be casted", v)
	}
	i = 4
	if diags := validateStringIsFloatRatio()(i, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%v should be an invalid float because it is an int out of range", v)
	}
}

func TestValidateDurationGeq0(t *testing.T) {
	v := "1ms"
	if diags := validateDurationGeq0()(v, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%v should be a valid durarion", v)
	}

	v = "-2h"
	if diags := validateDurationGeq0()(v, *new(cty.Path)); len(diags) == 0 {
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
	if diags := validateStringIsBase64Encoded()(v, *new(cty.Path)); len(diags) != 0 {
		t.Fatalf("%q should be base64 decodeable", v)
	}

	v = `%&df#3NkMjM4MQ==`
	if diags := validateStringIsBase64Encoded()(v, *new(cty.Path)); len(diags) == 0 {
		t.Fatalf("%q should NOT be base64 decodeable", v)
	}
}
