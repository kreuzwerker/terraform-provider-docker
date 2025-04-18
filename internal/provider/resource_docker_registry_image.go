package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the lifecycle of docker image in a registry. You can upload images to a registry (= `docker push`) and also delete them again",

		CreateContext: resourceDockerRegistryImageCreate,
		ReadContext:   resourceDockerRegistryImageRead,
		DeleteContext: resourceDockerRegistryImageDelete,
		UpdateContext: resourceDockerRegistryImageUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image.",
				Required:    true,
				ForceNew:    true,
			},

			"keep_remotely": {
				Type:        schema.TypeBool,
				Description: "If true, then the Docker image won't be deleted on destroy operation. If this is false, it will delete the image from the docker registry on destroy operation. Defaults to `false`",
				Default:     false,
				Optional:    true,
			},

			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Description: "If `true`, the verification of TLS certificates of the server/registry is disabled. Defaults to `false`",
				Optional:    true,
				Default:     false,
			},

			"triggers": {
				Description: "A map of arbitrary strings that, when changed, will force the `docker_registry_image` resource to be replaced. This can be used to repush a local image",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
			},

			"sha256_digest": {
				Type:        schema.TypeString,
				Description: "The sha256 digest of the image.",
				Computed:    true,
			},

			"auth_config": {
				Type:        schema.TypeList,
				Description: "Authentication configuration for the Docker registry. It is only used for this resource.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Description: "The address of the Docker registry.",
							Required:    true,
						},
						"username": {
							Type:        schema.TypeString,
							Description: "The username for the Docker registry.",
							Required:    true,
						},
						"password": {
							Type:        schema.TypeString,
							Description: "The password for the Docker registry.",
							Required:    true,
							Sensitive:   true,
						},
					},
				},
			},
		},
	}
}
