package docker

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDockerNetworkRead,

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
				Computed: true,
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
			},

			"scope": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

type ipamMap map[string]interface{}

func dataSourceDockerNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	name, nameOk := d.GetOk("name")
	_, idOk := d.GetOk("id")

	if !nameOk && !idOk {
		return diag.Errorf("One of id or name must be assigned")
	}

	client := meta.(*ProviderConfig).DockerClient

	network, err := client.NetworkInspect(ctx, name.(string), types.NetworkInspectOptions{})

	if err != nil {
		return diag.Errorf("Could not find docker network: %s", err)
	}

	d.SetId(network.ID)
	d.Set("name", network.Name)
	d.Set("scope", network.Scope)
	d.Set("driver", network.Driver)
	d.Set("options", network.Options)
	d.Set("internal", network.Internal)
	ipam := make([]ipamMap, len(network.IPAM.Config))
	for i, config := range network.IPAM.Config {
		ipam[i] = ipamMap{
			"subnet":      config.Subnet,
			"gateway":     config.Gateway,
			"aux_address": config.AuxAddress,
			"ip_range":    config.IPRange,
		}
	}
	err = d.Set("ipam_config", ipam)

	return nil
}
