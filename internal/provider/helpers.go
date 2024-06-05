package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func labelToPair(label map[string]interface{}) (string, string) {
	return label["label"].(string), label["value"].(string)
}

func labelSetToMap(labels *schema.Set) map[string]string {
	labelsSlice := labels.List()

	mapped := make(map[string]string, len(labelsSlice))
	for _, label := range labelsSlice {
		l, v := labelToPair(label.(map[string]interface{}))
		mapped[l] = v
	}
	return mapped
}

func hashLabel(v interface{}) int {
	labelMap := v.(map[string]interface{})
	return hashStringLabel(labelMap["label"].(string))
}

func hashStringLabel(str string) int {
	return schema.HashString(str)
}

func mapStringInterfaceToLabelList(labels map[string]interface{}) []interface{} {
	var mapped []interface{}
	for k, v := range labels {
		mapped = append(mapped, map[string]interface{}{
			"label": k,
			"value": fmt.Sprintf("%v", v),
		})
	}
	return mapped
}

func mapToLabelSet(labels map[string]string) *schema.Set {
	var mapped []interface{}
	for k, v := range labels {
		mapped = append(mapped, map[string]interface{}{
			"label": k,
			"value": v,
		})
	}
	return schema.NewSet(hashLabel, mapped)
}

var labelSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"label": {
			Type:        schema.TypeString,
			Description: "Name of the label",
			Required:    true,
			ForceNew:    true,
		},
		"value": {
			Type:        schema.TypeString,
			Description: "Value of the label",
			Required:    true,
			ForceNew:    true,
		},
	},
}

// gatherImmediateSubkeys given an incomplete attribute identifier, find all
// the strings (if any) that appear after this one in the various dot-separated
// identifiers.
func gatherImmediateSubkeys(attrs map[string]string, partialKey string) []string {
	immediateSubkeys := []string{}
	for k := range attrs {
		prefix := partialKey + "."
		if strings.HasPrefix(k, prefix) {
			rest := strings.TrimPrefix(k, prefix)
			parts := strings.SplitN(rest, ".", 2)
			immediateSubkeys = append(immediateSubkeys, parts[0])
		}
	}

	return immediateSubkeys
}

func getLabelMapForPartialKey(attrs map[string]string, partialKey string) map[string]string {
	setIDs := gatherImmediateSubkeys(attrs, partialKey)

	labelMap := map[string]string{}
	for _, id := range setIDs {
		if id == "#" {
			continue
		}
		prefix := partialKey + "." + id
		labelMap[attrs[prefix+".label"]] = attrs[prefix+".value"]
	}

	return labelMap
}

func testCheckLabelMap(name string, partialKey string, expectedLabels map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		attrs := s.RootModule().Resources[name].Primary.Attributes
		labelMap := getLabelMapForPartialKey(attrs, partialKey)

		if len(labelMap) != len(expectedLabels) {
			return fmt.Errorf("expected %v labels, found %v", len(expectedLabels), len(labelMap))
		}

		for l, v := range expectedLabels {
			if labelMap[l] != v {
				return fmt.Errorf("expected value %v for label %v, got %v", v, l, labelMap[v])
			}
		}

		return nil
	}
}

// containsIgnorableErrorMessage checks if the error message contains one of the
// message to ignore. Returns true if so, false otherwise (also if no ignorable message is given)
func containsIgnorableErrorMessage(errorMsg string, ignorableErrorMessages ...string) bool {
	for _, ignorableErrorMessage := range ignorableErrorMessages {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(ignorableErrorMessage)) {
			return true
		}
	}

	return false
}
