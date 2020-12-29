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
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"destroy_option": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"force": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}
