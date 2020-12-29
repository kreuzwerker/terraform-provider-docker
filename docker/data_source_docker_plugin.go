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

var errDataSourceKeyIsMissing = errors.New("One of id or alias must be assigned")

func getDataSourcePluginKey(d *schema.ResourceData) (string, error) {
	id, idOK := d.GetOk("id")
	alias, aliasOK := d.GetOk("alias")
	if idOK {
		if aliasOK {
			return "", errDataSourceKeyIsMissing
		}
		return id.(string), nil
	}
	if aliasOK {
		return alias.(string), nil
	}
	return "", errDataSourceKeyIsMissing
}

func dataSourceDockerPluginRead(d *schema.ResourceData, meta interface{}) error {
	key, err := getDataSourcePluginKey(d)
	if err != nil {
		return err
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
