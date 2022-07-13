package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_network` provides details about a specific Docker Network.",

		CreateContext: resourceDockerNetworkCreate,
		ReadContext:   resourceDockerNetworkRead,
		DeleteContext: resourceDockerNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker network.",
				Required:    true,
				ForceNew:    true,
			},

			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
				Elem:        labelSchema,
			},

			"check_duplicate": {
				Type:        schema.TypeBool,
				Description: "Requests daemon to check for networks with same name.",
				Optional:    true,
				ForceNew:    true,
			},

			"driver": {
				Type:        schema.TypeString,
				Description: "The driver of the Docker network. Possible values are `bridge`, `host`, `overlay`, `macvlan`. See [network docs](https://docs.docker.com/network/#network-drivers) for more details.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},

			"options": {
				Type:        schema.TypeMap,
				Description: "Only available with bridge networks. See [bridge options docs](https://docs.docker.com/engine/reference/commandline/network_create/#bridge-driver-options) for more details.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},

			"internal": {
				Type:        schema.TypeBool,
				Description: "Whether the network is internal.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},

			"attachable": {
				Type:        schema.TypeBool,
				Description: "Enable manual container attachment to the network.",
				Optional:    true,
				ForceNew:    true,
			},

			"ingress": {
				Type:        schema.TypeBool,
				Description: "Create swarm routing-mesh network. Defaults to `false`.",
				Optional:    true,
				ForceNew:    true,
			},

			"ipv6": {
				Type:        schema.TypeBool,
				Description: "Enable IPv6 networking. Defaults to `false`.",
				Optional:    true,
				ForceNew:    true,
			},

			"ipam_driver": {
				Type:        schema.TypeString,
				Description: "Driver used by the custom IP scheme of the network. Defaults to `default`",
				Default:     "default",
				Optional:    true,
				ForceNew:    true,
			},

			"ipam_config": {
				Type:        schema.TypeSet,
				Description: "The IPAM configuration options",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				// DiffSuppressFunc: suppressIfIPAMConfigWithIpv6Changes(),
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
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerNetworkV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return replaceLabelsMapFieldWithSetField(rawState), nil
				},
			},
		},
	}
}
