package provider

func replaceLabelsMapFieldWithSetField(rawState map[string]interface{}) map[string]interface{} {
	labelMapIFace := rawState["labels"]
	if labelMapIFace != nil {
		labelMap := labelMapIFace.(map[string]interface{})
		rawState["labels"] = mapStringInterfaceToLabelList(labelMap)
	} else {
		rawState["labels"] = []interface{}{}
	}

	return rawState
}

func migrateContainerLabels(rawState map[string]interface{}) map[string]interface{} {
	replaceLabelsMapFieldWithSetField(rawState)

	m, ok := rawState["mounts"]
	if !ok || m == nil {
		// https://github.com/terraform-providers/terraform-provider-docker/issues/264
		rawState["mounts"] = []interface{}{}
		return rawState
	}

	mounts := m.([]interface{})
	newMounts := make([]interface{}, len(mounts))
	for i, mountI := range mounts {
		mount := mountI.(map[string]interface{})
		volumeOptionsList := mount["volume_options"].([]interface{})

		if len(volumeOptionsList) != 0 {
			replaceLabelsMapFieldWithSetField(volumeOptionsList[0].(map[string]interface{}))
		}
		newMounts[i] = mount
	}
	rawState["mounts"] = newMounts

	return rawState
}

func migrateServiceLabels(rawState map[string]interface{}) map[string]interface{} {
	replaceLabelsMapFieldWithSetField(rawState)

	taskSpec := rawState["task_spec"].([]interface{})[0].(map[string]interface{})
	containerSpec := taskSpec["container_spec"].([]interface{})[0].(map[string]interface{})
	migrateContainerLabels(containerSpec)

	return rawState
}
