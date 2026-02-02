package provider

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dockerRegistryImageCreateDefaultTimeout = 20 * time.Minute
	dockerRegistryImageUpdateDefaultTimeout = 20 * time.Minute
	dockerRegistryImageDeleteDefaultTimeout = 20 * time.Minute
)

func resourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the lifecycle of docker image in a registry. You can upload images to a registry (= `docker push`) and also delete them again",

		CreateContext: resourceDockerRegistryImageCreate,
		ReadContext:   resourceDockerRegistryImageRead,
		DeleteContext: resourceDockerRegistryImageDelete,
		UpdateContext: resourceDockerRegistryImageUpdate,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(dockerRegistryImageCreateDefaultTimeout),
			Update: schema.DefaultTimeout(dockerRegistryImageUpdateDefaultTimeout),
			Delete: schema.DefaultTimeout(dockerRegistryImageDeleteDefaultTimeout),
		},

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

			"pull_by_digest": {
				Type: schema.TypeString,
				Description: `The computed image name and sha256 digest put together.

This will be something like ` + "`your-registry.tld/your-image@sha256:2e863c44b718727c860746568e1d54afd13b2fa71b160f5cd9058fc436217b30`" + `.

This value is reliable to prevent common issues, such as:

* downstream resources not being re-computed due to a tag name not changing
* downstream resources being recomputed, even if the docker image didn't change at all`,
				Computed: true,
			},

			"auth_config": AuthConfigSchema,
			"build": {
				Type:        schema.TypeSet,
				Description: "Configuration to build an image. Requires the `Use containerd for pulling and storing images` option to be disabled in the Docker Host(https://github.com/kreuzwerker/terraform-provider-docker/issues/534). Please see [docker build command reference](https://docs.docker.com/engine/reference/commandline/build/#options) too.",
				Optional:    true,
				MaxItems:    1,
				Elem:        buildSchema,
			},
		},
	}
}
