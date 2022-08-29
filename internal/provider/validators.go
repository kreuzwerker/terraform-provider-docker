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

func validateIntegerGeqThan(threshold int) schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(int)
		var diags diag.Diagnostics
		if value < threshold {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' cannot be lower than %d", value, threshold),
				Detail:   fmt.Sprintf("'%v' cannot be lower than %d", value, threshold),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

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
					Summary:  fmt.Sprintf("'%v' is not a float", v),
					Detail:   fmt.Sprintf("'%v' is not a float", v),
				}
				diags = append(diags, diag)
			}
			if value < 0.0 || value > 1.0 {
				diag := diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("'%v' has to be between 0.0 and 1.0", v),
					Detail:   fmt.Sprintf("'%v' has to be between 0.0 and 1.0", v),
				}
				diags = append(diags, diag)
			}
		case int:
			value := float64(t)
			if value < 0.0 || value > 1.0 {
				diag := diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("'%v' has to be between 0.0 and 1.0", v),
					Detail:   fmt.Sprintf("'%v' has to be between 0.0 and 1.0", v),
				}
				diags = append(diags, diag)
			}
		default:
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' is not a string", v),
				Detail:   fmt.Sprintf("'%v' is not a string", v),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

func validateDurationGeq0() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		dur, err := time.ParseDuration(value)
		if err != nil {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' is not a valid duration", value),
				Detail:   fmt.Sprintf("'%v' is not a valid duration", value),
			}
			diags = append(diags, diag)
		}
		if dur < 0 {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' duration must not be negative", value),
				Detail:   fmt.Sprintf("'%v' duration must not be negative", value),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

func validateStringMatchesPattern(pattern string) schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		compiledRegex, err := regexp.Compile(pattern)
		var diags diag.Diagnostics
		if err != nil {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("regex does not compile for pattern '%s'", pattern),
				Detail:   fmt.Sprintf("regex does not compile for pattern '%s'", pattern),
			}
			diags = append(diags, diag)
			return diags
		}

		value := v.(string)
		if !compiledRegex.MatchString(value) {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' doesn't match the pattern '%s'", value, pattern),
				Detail:   fmt.Sprintf("'%v' doesn't match the pattern '%s'", value, pattern),
			}
			diags = append(diags, diag)
		}

		return diags
	}
}

func validateStringIsBase64Encoded() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if _, err := base64.StdEncoding.DecodeString(value); err != nil {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' is not base64 decodeable", value),
				Detail:   fmt.Sprintf("'%v' is not base64 decodeable", value),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

func validateDockerContainerPath() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if !regexp.MustCompile(`^[a-zA-Z]:[\\/]|^/`).MatchString(value) {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("'%v' must be an absolute path", value),
				Detail:   fmt.Sprintf("'%v' must be an absolute path", value),
			}
			diags = append(diags, diag)
		}

		return diags
	}
}
