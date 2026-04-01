package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the lifecycle of a Docker plugin.",

		CreateContext: resourceDockerPluginCreate,
		ReadContext:   resourceDockerPluginRead,
		UpdateContext: resourceDockerPluginUpdate,
		DeleteContext: resourceDockerPluginDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Description:      "Docker Plugin name",
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: diffSuppressFuncPluginName,
				ValidateFunc:     validateFuncPluginName,
			},
			"alias": {
				Type:        schema.TypeString,
				Description: "Docker Plugin alias",
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					return complementTag(oldV) == complementTag(newV)
				},
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "If `true` the plugin is enabled. Defaults to `true`",
				Default:     true,
				Optional:    true,
			},
			"grant_all_permissions": {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "If true, grant all permissions necessary to run the plugin",
				ConflictsWith: []string{"grant_permissions"},
			},
			"grant_permissions": {
				Type:          schema.TypeSet,
				Description:   "Grant specific permissions only",
				Optional:      true,
				ConflictsWith: []string{"grant_all_permissions"},
				Set:           dockerPluginGrantPermissionsSetFunc,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the permission",
							Required:    true,
						},
						"value": {
							Type:        schema.TypeSet,
							Description: "The value of the permission",
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"env": {
				Type:        schema.TypeSet,
				Description: "The environment variables in the form of `KEY=VALUE`, e.g. `DEBUG=0`",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"plugin_reference": {
				Type:        schema.TypeString,
				Description: "Docker Plugin Reference",
				Computed:    true,
			},

			"force_destroy": {
				Type:        schema.TypeBool,
				Description: "If true, then the plugin is destroyed forcibly",
				Optional:    true,
			},
			"enable_timeout": {
				Type:        schema.TypeInt,
				Description: "HTTP client timeout to enable the plugin",
				Optional:    true,
			},
			"force_disable": {
				Type:        schema.TypeBool,
				Description: "If true, then the plugin is disabled forcibly",
				Optional:    true,
			},
		},
	}
}
