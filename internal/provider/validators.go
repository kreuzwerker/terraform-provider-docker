package provider

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//nolint:staticcheck
func validateIntegerInRange(min, max int) schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(int)
		var diags diag.Diagnostics
		if value < min {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%q is lower than %d", value, min),
				Detail:   fmt.Sprintf("%q is lower than %d", value, min),
			}
			diags = append(diags, diag)
		}
		if value > max {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%q is greater than %d", value, max),
				Detail:   fmt.Sprintf("%q is greater than %d", value, max),
			}
			diags = append(diags, diag)
		}

		return diags
	}
}

//nolint:staticcheck
func validateIntegerGeqThan(threshold int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < threshold {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d", k, threshold))
		}
		return
	}
}

//nolint:staticcheck
func validateFloatRatio() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(float64)
		if value < 0.0 || value > 1.0 {
			errors = append(errors, fmt.Errorf(
				"%q has to be between 0.0 and 1.0", k))
		}
		return
	}
}

//nolint:staticcheck
func validateStringIsFloatRatio() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		switch t := v.(type) {
		case string:
			stringValue := t
			value, err := strconv.ParseFloat(stringValue, 64)
			if err != nil {
				errors = append(errors, fmt.Errorf(
					"%q is not a float", k))
			}
			if value < 0.0 || value > 1.0 {
				errors = append(errors, fmt.Errorf(
					"%q has to be between 0.0 and 1.0", k))
			}
		case int:
			value := float64(t)
			if value < 0.0 || value > 1.0 {
				errors = append(errors, fmt.Errorf(
					"%q has to be between 0.0 and 1.0", k))
			}
		default:
			errors = append(errors, fmt.Errorf(
				"%q is not a string", k))
		}
		return
	}
}

//nolint:staticcheck
func validateDurationGeq0() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		dur, err := time.ParseDuration(value)
		if err != nil {
			errors = append(errors, fmt.Errorf(
				"%q is not a valid duration", k))
		}
		if dur < 0 {
			errors = append(errors, fmt.Errorf(
				"duration must not be negative"))
		}
		return
	}
}

//nolint:staticcheck
func validateStringMatchesPattern(pattern string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		compiledRegex, err := regexp.Compile(pattern)
		if err != nil {
			errors = append(errors, fmt.Errorf(
				"%q regex does not compile", pattern))
			return
		}

		value := v.(string)
		if !compiledRegex.MatchString(value) {
			errors = append(errors, fmt.Errorf(
				"%q doesn't match the pattern (%q): %q",
				k, pattern, value))
		}

		return
	}
}

func validateStringIsBase64Encoded() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if _, err := base64.StdEncoding.DecodeString(value); err != nil {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%q is not base64 decodeable", value),
				Detail:   fmt.Sprintf("%q is not base64 decodeable", value),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

func validateDockerContainerPath(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z]:\\|^/`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be an absolute path", k))
	}

	return
}
