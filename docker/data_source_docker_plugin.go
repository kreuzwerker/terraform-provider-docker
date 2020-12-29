package docker

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceDockerPlugin() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDockerPluginRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"alias": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Docker Plugin alias.",
			},

			"plugin_reference": {
				Type:        schema.TypeString,
				Description: "Docker Plugin Reference.",
				Optional:    true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"grant_all_permissions": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, grant all permissions necessary to run the plugin",
			},
			"args": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func getDataSourcePluginKey(d *schema.ResourceData) string {
	if id, ok := d.GetOk("id"); ok {
		return id.(string)
	}
	if alias, ok := d.GetOk("alias"); ok {
		return alias.(string)
	}
	return ""
}

var errDataSourceKeyIsMissing = errors.New("One of id or alias must be assigned")

func dataSourceDockerPluginRead(d *schema.ResourceData, meta interface{}) error {
	key := getDataSourcePluginKey(d)
	if key == "" {
		return errDataSourceKeyIsMissing
	}
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	plugin, _, err := client.PluginInspectWithRaw(ctx, key)
	if err != nil {
		return fmt.Errorf("inspect a Docker plugin "+key+": %w", err)
	}

	setDockerPlugin(d, plugin)
	return nil
}
