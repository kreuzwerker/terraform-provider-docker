package docker

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

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
