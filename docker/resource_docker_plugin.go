package docker

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerPlugin() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerPluginCreate,
		Read:   resourceDockerPluginRead,
		Update: resourceDockerPluginUpdate,
		Delete: resourceDockerPluginDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"plugin_reference": {
				Type:        schema.TypeString,
				Description: "Docker Plugin Reference",
				Required:    true,
				ForceNew:    true,
			},
			"alias": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: "Docker Plugin alias",
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					return complementTag(oldV) == complementTag(newV)
				},
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"grant_all_permissions": {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "If true, grant all permissions necessary to run the plugin",
				ConflictsWith: []string{"grant_permissions"},
			},
			"grant_permissions": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"grant_all_permissions"},
				Set:           dockerPluginGrantPermissionsSetFunc,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"enable_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "HTTP client timeout to enable the plugin",
			},
			"force_disable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, then the plugin is disabled forcibly when the plugin is disabled",
			},
		},
	}
}
