package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
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
			"repo_digest": {
				Type:        schema.TypeString,
				Description: "The image sha256 digest in the form of `repo[:tag]@sha256:<hash>`.",
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

	repoDigest, err := determineRepoDigest(imageName, foundImage, foundImage.ID)
	if err != nil {
		return diag.Errorf("did not determine docker image repo digest for image name '%s' - repo digests: %v", imageName, foundImage.RepoDigests)
	}

	d.SetId(foundImage.ID)
	d.Set("name", imageName)
	d.Set("latest", foundImage.ID)
	d.Set("repo_digest", repoDigest)

	return nil
}

func determineRepoDigest(imageName string, imageToQuery *types.ImageSummary, fallbackDigest string) (string, error) {
	if len(imageToQuery.RepoDigests) == 1 {
		return imageToQuery.RepoDigests[0], nil
	}

	for _, repoDigest := range imageToQuery.RepoDigests {
		if strings.Contains(repoDigest, imageName) {
			return repoDigest, nil
		}
	}

	if fallbackDigest != "" {
		return fallbackDigest, nil
	}

	return "", fmt.Errorf("could not determine repo digest for imageName. Fallback was empty as well")
}
