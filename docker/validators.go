package docker

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func validateIntegerInRange(min, max int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < min {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d: %d", k, min, value))
		}
		if value > max {
			errors = append(errors, fmt.Errorf(
				"%q cannot be higher than %d: %d", k, max, value))
		}
		return
	}
}

func validateIntegerGeqThan(threshold int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < threshold {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %q", k, threshold))
		}
		return
	}
}

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

func validateStringIsFloatRatio() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		switch v.(type) {
		case string:
			stringValue := v.(string)
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
			value := float64(v.(int))
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

func validateStringIsBase64Encoded() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		if _, err := base64.StdEncoding.DecodeString(value); err != nil {
			errors = append(errors, fmt.Errorf(
				"%q is not base64 decodeable", k))
		}

		return
	}
}

func validateDockerContainerPath(v interface{}, k string) (ws []string, errors []error) {

	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z]:\\|^/`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be an absolute path", k))
	}

	return
}
