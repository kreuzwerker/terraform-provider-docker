package provider

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
)

func dataSourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_container` provides details about a specific Docker Container that exists on the host",

		ReadContext: dataSourceDockerContainerRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Filter value for finding a container by name. Multiple results will return an error.",
				Optional:    true,
				Computed:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the container",
				Optional:    true,
			},
			"status": {
				Type:        schema.TypeString,
				Description: "The status of the container",
				Computed:    true,
			},
			"image": {
				Type:        schema.TypeString,
				Description: "The image used to create the container",
				Computed:    true,
			},
			"created": {
				Type:        schema.TypeString,
				Description: "When the container was created",
				Computed:    true,
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
	filters := filters.NewArgs()
	if name != ""  {
		filters.Add("name", name.(string))
	}
	if id != "" {
		filters.Add("id", id.(string))
	}
	options := types.ContainerListOptions{
		Filters: filters,
	}
	containers, containerErr := client.ContainerList(ctx, options)
	if containerErr != nil {
		return diag.Errorf("Error reading docker containers")
	}
	if len(containers) > 1 {
		return diag.Errorf("Found multiple containers with the filter: %v", filters)
	}
	if len(containers) == 0 {
		return diag.Errorf("Could not find the docker container from filter: %v", filters)
	}
	id = containers[0].ID
	container, err := client.ContainerInspect(ctx, id.(string))

	jsonObj, _ := json.MarshalIndent(container, "", "\t")
	log.Printf("[DEBUG] Docker container inspect from stateFunc: %s", jsonObj)

	d.SetId(id.(string))
	d.Set("name", container.Name)
	d.Set("status", container.State.Status)
	d.Set("created", container.Created)
	d.Set("image", container.Image)

	if err != nil {
		return diag.Errorf("Could not read docker container ID %s", id.(string))
	}

	return nil
}
