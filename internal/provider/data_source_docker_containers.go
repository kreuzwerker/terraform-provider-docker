package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerContainers() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_containers` provides details about existing containers.",

		ReadContext: dataSourceDockerContainersRead,

		Schema: map[string]*schema.Schema{
			"containers": {
				Type:        schema.TypeList,
				Description: "The list of JSON-encoded containers currently running on the docker host.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceDockerContainersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	containers, err := client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return diag.Errorf("Could not list docker containers: %v", err)
	}

	var jsons []string
	for _, container := range containers {
		json, err := json.Marshal(container)
		if err != nil {
			return diag.Errorf("Could not marshal JSON from container ID %s: %v", container.ID, err)
		}
		jsons = append(jsons, string(json))
	}

	// Sort the JSON strings to ensure consistent order for hash calculation
	sort.Strings(jsons)
	if err := d.Set("containers", jsons); err != nil {
		return diag.Errorf("Failed to set containers: %v", err)
	}

	md5sum := md5.Sum([]byte(strings.Join(jsons, "")))
	digest := hex.EncodeToString(md5sum[:])
	d.SetId(digest)

	return nil
}
