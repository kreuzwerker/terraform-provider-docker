package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the local Docker plugin. The plugin must be installed locally.",

		Read: dataSourceDockerPluginRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the plugin, which has precedence over the `alias` of both are given",
				Optional:    true,
			},
			"alias": {
				Type:        schema.TypeString,
				Description: "The alias of the Docker plugin. If the tag is omitted, `:latest` is complemented to the attribute value.",
				Optional:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The plugin name. If the tag is omitted, `:latest` is complemented to the attribute value.",
				Computed:    true,
			},
			"plugin_reference": {
				Type:        schema.TypeString,
				Description: "The Docker Plugin Reference",
				Computed:    true,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "If `true` the plugin is enabled",
				Computed:    true,
			},
			"grant_all_permissions": {
				Type:        schema.TypeBool,
				Description: "If true, grant all permissions necessary to run the plugin",
				Computed:    true,
			},
			"env": {
				Type:        schema.TypeSet,
				Description: "The environment variables in the form of `KEY=VALUE`, e.g. `DEBUG=0`",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

var errDataSourceKeyIsMissing = errors.New("one of id or alias must be assigned")

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
	ctx := context.Background()
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return err
	}
	plugin, _, err := client.PluginInspectWithRaw(ctx, key)
	if err != nil {
		return fmt.Errorf("inspect a Docker plugin "+key+": %w", err)
	}

	setDockerPlugin(d, plugin)
	return nil
}
