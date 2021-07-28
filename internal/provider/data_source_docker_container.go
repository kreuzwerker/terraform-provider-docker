package provider

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_container` provides details about a specific Docker Container that exists on the host",

		ReadContext: dataSourceDockerContainerRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the container",
				Optional:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the container",
				Optional:    true,
			},
		},
	}
}

func dataSourceDockerContainerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name, nameOk := d.GetOk("name")
	id, idOk := d.GetOk("id")

	if !nameOk && !idOk {
		return diag.Errorf("One of id or name must be assigned")
	}

	client := meta.(*ProviderConfig).DockerClient
	if id == "" {
		filters := filters.NewArgs()
		filters.Add("name", name.(string))
		options := types.ContainerListOptions{
			Filters: filters,
		}
		containers, containerErr := client.ContainerList(ctx, options)
		if containerErr != nil {
			diag.Errorf("Error reading docker containers")
		}
		if len(containers) > 1 {
			diag.Errorf("Found multiple containers with the name: %s", name)
		}
		if len(containers) == 0 {
			diag.Errorf("Could not find the docker container: %s", name)
		}
		id = containers[0].ID
	}
	d.SetId(id.(string))
	err := resourceDockerContainerRead(ctx, d, meta)

	if err != nil {
		return diag.Errorf("Could not read docker container ID %s", id.(string))
	}

	return nil
}
