package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func resourceDockerTag() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a docker tag. It has the exact same functionality as the `docker tag` command. Deleting the resource will neither delete the source nor target images. The source image must exist on the machine running the docker daemon.",
		CreateContext: resourceDockerTagCreate,
		DeleteContext: resourceDockerTagDelete,
		ReadContext:   resourceDockerTagRead,

		Schema: map[string]*schema.Schema{
			"source_image": {
				Type:        schema.TypeString,
				Description: "Name of the source image.",
				Required:    true,
				ForceNew:    true,
			},
			"source_image_id": {
				Type:        schema.TypeString,
				Description: "ImageID of the source image in the format of `sha256:<<ID>>`",
				Computed:    true,
			},
			"target_image": {
				Type:        schema.TypeString,
				Description: "Name of the target image.",
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}
