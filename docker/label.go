package docker

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
		"label": &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of the label",
			Required:    true,
		},
		"value": &schema.Schema{
			Type:        schema.TypeString,
			Description: "Value of the label",
			Required:    true,
		},
	},
}

//gatherImmediateSubkeys given an incomplete attribute identifier, find all
//the strings (if any) that appear after this one in the various dot-separated
//identifiers.
func gatherImmediateSubkeys(attrs map[string]string, partialKey string) []string {
	var immediateSubkeys = []string{}
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

	var labelMap = map[string]string{}
	for _, id := range setIDs {
		prefix := partialKey + "." + id
		labelMap[attrs[prefix+".label"]] = attrs[prefix+".value"]
	}

	return labelMap
}
