package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDockerNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"driver": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"options": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},

			"internal": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"ipam_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_range": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"gateway": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"aux_address": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerIpamConfigHash,
			},

			"scope": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDockerNetworkRead(d *schema.ResourceData, meta interface{}) error {

	name, nameOk := d.GetOk("name")
	_, idOk := d.GetOk("id")

	if !nameOk && !idOk {
		return fmt.Errorf("One of id or name must be assigned")
	}

	client := meta.(*ProviderConfig).DockerClient

	network, err := client.NetworkInspect(context.Background(), name.(string), types.NetworkInspectOptions{})

	if err != nil {
		return fmt.Errorf("Could not find docker network: %s", err)
	}

	d.SetId(network.ID)
	d.Set("name", network.Name)
	d.Set("scope", network.Scope)
	d.Set("driver", network.Driver)
	d.Set("options", network.Options)
	d.Set("internal", network.Internal)
	d.Set("imap_config", network.IPAM)

	return nil
}
