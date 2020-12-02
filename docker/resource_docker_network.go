package docker

import (
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerNetworkCreate,
		Read:   resourceDockerNetworkRead,
		Delete: resourceDockerNetworkDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"labels": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     labelSchema,
			},

			"check_duplicate": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"options": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"internal": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"attachable": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ingress": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipam_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "default",
			},

			"ipam_config": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				// DiffSuppressFunc: suppressIfIPAMConfigWithIpv6Changes(),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_range": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"gateway": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"aux_address": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerNetworkV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return replaceLabelsMapFieldWithSetField(rawState), nil
				},
			},
		},
	}
}

func resourceDockerNetworkV0() *schema.Resource {
	return &schema.Resource{
		// This is only used for state migration, so the CRUD
		// callbacks are no longer relevant
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"check_duplicate": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "bridge",
			},

			"options": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"internal": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"attachable": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ingress": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipam_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "default",
			},

			"ipam_config": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				// DiffSuppressFunc: suppressIfIPAMConfigWithIpv6Changes(),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_range": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"gateway": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"aux_address": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func suppressIfIPAMConfigWithIpv6Changes() schema.SchemaDiffSuppressFunc {
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
