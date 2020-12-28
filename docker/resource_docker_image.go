package docker

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerImageCreate,
		Read:   resourceDockerImageRead,
		Update: resourceDockerImageUpdate,
		Delete: resourceDockerImageDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"latest": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"keep_locally": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"pull_trigger": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"pull_triggers"},
				Deprecated:    "Use field pull_triggers instead",
			},

			"pull_triggers": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"output": {
				Type:     schema.TypeString,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"force_remove": {
				Type:        schema.TypeBool,
				Description: "Force remove the image when the resource is destroyed",
				Optional:    true,
			},

			"build": {
				Type:          schema.TypeSet,
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
							Description: "Name of the Dockerfile (Default is 'PATH/Dockerfile')",
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
							Description: "Remove intermediate containers after a successful build (default true)",
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
