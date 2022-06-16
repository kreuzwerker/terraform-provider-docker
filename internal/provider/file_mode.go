package provider

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const defaultFileMode = "0444"

func convertStrToFileMode(s string) (os.FileMode, error) {
	if s == "" {
		s = defaultFileMode
	}
	a, err := strconv.ParseInt(s, 8, 32)
	if err != nil {
		return 0, err
	}
	if a < 0 {
		return 0, fmt.Errorf("file_mode must be greater equal than 0: %d", a)
	}
	if a > 0o4777 {
		return 0, fmt.Errorf("file_mode must be less equal than 0o4777: %d", a)
	}
	return os.FileMode(a), nil
}

func fileModeDiffSuppressFunc(k, oldV, newV string, d *schema.ResourceData) bool {
	log.Printf("[DEBUG] DiffSuppressFunc(key: %s, old value: %s, new value: %s)", k, oldV, newV)
	if oldV == newV {
		return true
	}
	a, err := convertStrToFileMode(oldV)
	if err != nil {
		log.Printf("[DEBUG] DiffSuppressFunc(key: %s, old value: %s, new value: %s): old value is invalid: %s", k, oldV, newV, err.Error())
		return false
	}
	b, err := convertStrToFileMode(newV)
	if err != nil {
		log.Printf("[DEBUG] DiffSuppressFunc(key: %s, old value: %s, new value: %s): new value is invalid: %s", k, oldV, newV, err.Error())
		return false
	}
	return a == b
}

func fileModeSchemaValidateDiagFunc(value interface{}, ctyPath cty.Path) diag.Diagnostics {
	if _, err := convertStrToFileMode(value.(string)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
