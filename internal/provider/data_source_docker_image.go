package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerImage() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_image` provides details about a specific Docker Image which need to be presend on the Docker Host",

		ReadContext: dataSourceDockerImageRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image, including any tags or SHA256 repo digests.",
				Required:    true,
			},
			"latest": {
				Type:        schema.TypeString,
				Description: "The ID of the image in the form of `sha256:<hash>` image digest. Do not confuse it with the default `latest` tag.",
				Computed:    true,
				Deprecated:  "Use `sha256_digest` instead",
			},
			"sha256_digest": {
				Type:        schema.TypeString,
				Description: "The image sha256 digest in the form of `sha256:<hash>`.",
				Computed:    true,
			},
		},
	}
}

func dataSourceDockerImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	var data Data
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return diag.Errorf("Error reading docker image list: %s", err)
	}
	for id := range data.DockerImages {
		log.Printf("[DEBUG] local images data: %v", id)
	}

	imageName := d.Get("name").(string)

	foundImage := searchLocalImages(ctx, client, data, imageName)
	if foundImage == nil {
		return diag.Errorf("did not find docker image '%s'", imageName)
	}

	d.SetId(foundImage.ID)
	d.Set("name", imageName)
	d.Set("latest", foundImage.ID)
	d.Set("sha256_digest", foundImage.ID)

	return nil
}
