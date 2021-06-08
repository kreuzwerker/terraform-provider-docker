package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerImage() *schema.Resource {
	return &schema.Resource{
		Description: "Pulls a Docker image to a given Docker host from a Docker Registry.\n This resource will *not* pull new layers of the image automatically unless used in conjunction with [docker_registry_image](registry_image.md) data source to update the `pull_triggers` field.",

		CreateContext: resourceDockerImageCreate,
		ReadContext:   resourceDockerImageRead,
		UpdateContext: resourceDockerImageUpdate,
		DeleteContext: resourceDockerImageDelete,

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
				Deprecated:  "Use sha256_digest instead",
			},

			"repo_digest": {
				Type:        schema.TypeString,
				Description: "The image sha256 digest in the form of `repo[:tag]@sha256:<hash>`.",
				Computed:    true,
			},

			"keep_locally": {
				Type:        schema.TypeBool,
				Description: "If true, then the Docker image won't be deleted on destroy operation. If this is false, it will delete the image from the docker local storage on destroy operation.",
				Optional:    true,
			},

			"pull_trigger": {
				Type:          schema.TypeString,
				Description:   "A value which cause an image pull when changed",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"pull_triggers"},
				Deprecated:    "Use field pull_triggers instead",
			},

			"pull_triggers": {
				Type:        schema.TypeSet,
				Description: "List of values which cause an image pull when changed. This is used to store the image digest from the registry when using the [docker_registry_image](../data-sources/registry_image.md).",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"output": {
				Type:       schema.TypeString,
				Deprecated: "Is unused and will be removed.",
				Computed:   true,
				Elem: &schema.Schema{
					Type:       schema.TypeString,
					Deprecated: "Is unused and will be removed.",
				},
			},

			"force_remove": {
				Type:        schema.TypeBool,
				Description: "If true, then the image is removed forcibly when the resource is destroyed.",
				Optional:    true,
			},

			"build": {
				Type:          schema.TypeSet,
				Description:   "Configuration to build an image. Please see [docker build command reference](https://docs.docker.com/engine/reference/commandline/build/#options) too.",
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"pull_triggers", "pull_trigger"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:        schema.TypeString,
							Description: "Context path",
							Required:    true,
							ForceNew:    true,
						},
						"dockerfile": {
							Type:        schema.TypeString,
							Description: "Name of the Dockerfile. Defaults to `Dockerfile`.",
							Optional:    true,
							Default:     "Dockerfile",
							ForceNew:    true,
						},
						"tag": {
							Type:        schema.TypeList,
							Description: "Name and optionally a tag in the 'name:tag' format",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"force_remove": {
							Type:        schema.TypeBool,
							Description: "Always remove intermediate containers",
							Optional:    true,
						},
						"remove": {
							Type:        schema.TypeBool,
							Description: "Remove intermediate containers after a successful build. Defaults to  `true`.",
							Default:     true,
							Optional:    true,
						},
						"no_cache": {
							Type:        schema.TypeBool,
							Description: "Do not use cache when building the image",
							Optional:    true,
						},
						"target": {
							Type:        schema.TypeString,
							Description: "Set the target build stage to build",
							Optional:    true,
						},
						"build_arg": {
							Type:        schema.TypeMap,
							Description: "Set build-time variables",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							ForceNew: true,
						},
						"label": {
							Type:        schema.TypeMap,
							Description: "Set metadata for an image",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}
