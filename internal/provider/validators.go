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
func validateIntegerGeqThan(threshold int) schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(int)
		var diags diag.Diagnostics
		if value < threshold {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%q cannot be lower than %d", value, threshold),
				Detail:   fmt.Sprintf("%q cannot be lower than %d", value, threshold),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

//nolint:staticcheck
func validateStringIsFloatRatio() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics
		switch t := v.(type) {
		case string:
			stringValue := t
			value, err := strconv.ParseFloat(stringValue, 64)
			if err != nil {
				diag := diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("%v is not a float", v),
					Detail:   fmt.Sprintf("%v is not a float", v),
				}
				diags = append(diags, diag)
			}
			if value < 0.0 || value > 1.0 {
				diag := diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("%v has to be between 0.0 and 1.0", v),
					Detail:   fmt.Sprintf("%v has to be between 0.0 and 1.0", v),
				}
				diags = append(diags, diag)
			}
		case int:
			value := float64(t)
			if value < 0.0 || value > 1.0 {
				diag := diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("%v has to be between 0.0 and 1.0", v),
					Detail:   fmt.Sprintf("%v has to be between 0.0 and 1.0", v),
				}
				diags = append(diags, diag)
			}
		default:
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%v is not a string", v),
				Detail:   fmt.Sprintf("%v is not a string", v),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

//nolint:staticcheck
func validateDurationGeq0() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics
		value := v.(string)
		dur, err := time.ParseDuration(value)
		if err != nil {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%v is not a valid duration", value),
				Detail:   fmt.Sprintf("%v is not a valid duration", value),
			}
			diags = append(diags, diag)
		}
		if dur < 0 {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("%v duration must not be negative", value),
				Detail:   fmt.Sprintf("%v duration must not be negative", value),
			}
			diags = append(diags, diag)
		}
		return diags
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
