package docker

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func replaceLabelsMapFieldWithSetField(rawState map[string]interface{}) map[string]interface{} {
	labelMap := rawState["labels"].(map[string]interface{})
	rawState["labels"] = mapStringInterfaceToLabelSet(labelMap)

	return rawState
}

func migrateContainerLabels(rawState map[string]interface{}) map[string]interface{} {
	rawState = replaceLabelsMapFieldWithSetField(rawState)

	mounts := rawState["mounts"].(*schema.Set).List()
	newMounts := make([]interface{}, len(mounts))
	for i, mountI := range newMounts {
		mount := mountI.(map[string]interface{})
		volumeOptions := mount["volume_options"].([]interface{})[0].(map[string]interface{})

		mount["volume_options"] = replaceLabelsMapFieldWithSetField(volumeOptions)
		newMounts[i] = mount
	}
	rawState["mounts"] = newMounts

	return rawState
}

func migrateServiceLabels(rawState map[string]interface{}) map[string]interface{} {
	rawState = replaceLabelsMapFieldWithSetField(rawState)

	taskSpec := rawState["task_spec"].([]interface{})[0].(map[string]interface{})
	containerSpec := taskSpec["container_spec"].([]interface{})[0].(map[string]interface{})
	taskSpec["container_spec"] = migrateContainerLabels(containerSpec)

	rawState["task_spec"] = taskSpec
	return rawState
}
