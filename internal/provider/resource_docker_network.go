package provider

import (
	"context"
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_network` provides a docker network resource.",

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
			"ipam_options": {
				Type:        schema.TypeMap,
				Description: "Provide explicit options to the IPAM driver. Valid options vary with `ipam_driver` and refer to that driver's documentation for more details.",
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

func suppressIfIPAMConfigWithIpv6Changes() schema.SchemaDiffSuppressFunc { //nolint:deadcode,unused
	return func(k, old, new string, d *schema.ResourceData) bool {
		// the initial case when the resource is created
		if old == "" && new != "" {
			return false
		}

		// if ipv6 is not given we do not consider
		ipv6, ok := d.GetOk("ipv6")
		if !ok {
			return false
		}

		// if ipv6 is given but false we do not consider
		isIPv6 := ipv6.(bool)
		if !isIPv6 {
			return false
		}
		if k == "ipam_config.#" {
			log.Printf("[INFO] ipam_config: k: %q, old: %s, new: %s\n", k, old, new)
			oldVal, _ := strconv.Atoi(string(old))
			newVal, _ := strconv.Atoi(string(new))
			log.Printf("[INFO] ipam_config: oldVal: %d, newVal: %d\n", oldVal, newVal)
			if newVal <= oldVal {
				log.Printf("[INFO] suppressingDiff for ipam_config: oldVal: %d, newVal: %d\n", oldVal, newVal)
				return true
			}
		}
		if regexp.MustCompile(`ipam_config\.\d+\.gateway`).MatchString(k) {
			ip := net.ParseIP(old)
			ipv4Address := ip.To4()
			log.Printf("[INFO] ipam_config.gateway: k: %q, old: %s, new: %s - %v\n", k, old, new, ipv4Address != nil)
			// is an ipv4Address and content changed from non-empty to empty
			if ipv4Address != nil && old != "" && new == "" {
				log.Printf("[INFO] suppressingDiff for ipam_config.gateway %q: oldVal: %s, newVal: %s\n", ipv4Address.String(), old, new)
				return true
			}
		}
		if regexp.MustCompile(`ipam_config\.\d+\.subnet`).MatchString(k) {
			if old != "" && new == "" {
				log.Printf("[INFO] suppressingDiff for ipam_config.subnet: oldVal: %s, newVal: %s\n", old, new)
				return true
			}
		}
		return false
	}
}
