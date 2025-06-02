package provider

import (
	"context"
	"log"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_network` provides details about a specific Docker Network.",

		ReadContext: dataSourceDockerNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker network.",
				Required:    true,
			},

			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"driver": {
				Type:        schema.TypeString,
				Description: "The driver of the Docker network. Possible values are `bridge`, `host`, `overlay`, `macvlan`. See [network docs](https://docs.docker.com/network/#network-drivers) for more details.",
				Computed:    true,
			},

			"options": {
				Type:        schema.TypeMap,
				Description: "Only available with bridge networks. See [bridge options docs](https://docs.docker.com/engine/reference/commandline/network_create/#bridge-driver-options) for more details.",
				Computed:    true,
			},

			"internal": {
				Type:        schema.TypeBool,
				Description: "If `true`, the network is internal.",
				Computed:    true,
			},

			"ipam_config": {
				Type:        schema.TypeSet,
				Description: "The IPAM configuration options",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:        schema.TypeString,
							Description: "The subnet in CIDR form",
							Optional:    true,
							ForceNew:    true,
						},

						"ip_range": {
							Type:        schema.TypeString,
							Description: "The ip range in CIDR form",
							Optional:    true,
							ForceNew:    true,
						},

						"gateway": {
							Type:        schema.TypeString,
							Description: "The IP address of the gateway",
							Optional:    true,
							ForceNew:    true,
						},

						"aux_address": {
							Type:        schema.TypeMap,
							Description: "Auxiliary IPv4 or IPv6 addresses used by Network driver",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"scope": {
				Type:        schema.TypeString,
				Description: "Scope of the network. One of `swarm`, `global`, or `local`.",
				Computed:    true,
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

	network, err := client.NetworkInspect(ctx, name.(string), network.InspectOptions{})
	if err != nil {
		return diag.Errorf("Could not find docker network: %s", err)
	}

	d.SetId(network.ID)
	d.Set("name", network.Name)
	d.Set("scope", network.Scope)
	d.Set("driver", network.Driver)
	d.Set("options", network.Options)
	d.Set("internal", network.Internal)
	if err = d.Set("ipam_config", flattenIpamConfig(network.IPAM.Config)); err != nil {
		log.Printf("[WARN] failed to set ipam config from API: %s", err)
	}

	return nil
}

func flattenIpamConfig(in []network.IPAMConfig) []ipamMap {
	ipam := make([]ipamMap, len(in))
	for i, config := range in {
		ipam[i] = ipamMap{
			"subnet":      config.Subnet,
			"gateway":     config.Gateway,
			"aux_address": config.AuxAddress,
			"ip_range":    config.IPRange,
		}
	}

	return ipam
}
