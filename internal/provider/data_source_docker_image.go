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
			"repo_digest": {
				Type:        schema.TypeString,
				Description: "The image sha256 digest in the form of `repo[:tag]@sha256:<hash>`. It may be empty in the edge case where the local image was pulled from a repo, tagged locally, and then referred to in the data source by that local name/tag.",
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

	repoDigest, err := determineRepoDigest(imageName, foundImage)
	if err != nil {
		return diag.Errorf("failed to determine repo digest for image data source: %v", err)
	}

	d.SetId(foundImage.ID)
	d.Set("name", imageName)
	d.Set("repo_digest", repoDigest)

	return nil
}

// determineRepoDigest determines the repo digest for a local image name
// see https://github.com/kreuzwerker/terraform-provider-docker/pull/212#discussion_r646025706 for details
func determineRepoDigest(imageName string, imageToQuery *types.ImageSummary) (string, error) {
	// the edge case where the local image was pulled from a repo, tagged locally,
	// and then referred to in the data source by that local name/tag...
	if len(imageToQuery.RepoDigests) == 0 {
		return "", nil
	}

	// the standard case when there is only one digest
	if len(imageToQuery.RepoDigests) == 1 {
		return imageToQuery.RepoDigests[0], nil
	}

	// the special case when the same image is in multiple registries
	// we first need to strip a possible tag because the digest do not contain it
	imageNameWithoutTag := imageName
	tagColonIndex := strings.Index(imageName, ":")
	// we have a tag
	if tagColonIndex != -1 {
		imageNameWithoutTag = imageName[:tagColonIndex]
	}

	for _, repoDigest := range imageToQuery.RepoDigests {
		// we look explicitly at the beginning of the digest
		// as the image name is e.g. nginx:1.19.1 or 127.0.0.1/nginx:1.19.1 and the digests are
		// "RepoDigests": [
		//     "127.0.0.1:15000/nginx@sha256:a5a1e8e5148de5ebc9697b64e4edb2296b5aac1f05def82efdc00ccb7b457171",
		//     "nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2"
		// ],
		if strings.HasPrefix(repoDigest, imageNameWithoutTag) {
			return repoDigest, nil
		}
	}

	return "", fmt.Errorf("could not determine repo digest for image name '%s' - repo digests: %v", imageName, imageToQuery.RepoDigests)
}
