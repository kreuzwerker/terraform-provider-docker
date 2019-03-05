package docker

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Create:        resourceDockerContainerCreate,
		Read:          resourceDockerContainerRead,
		Update:        resourceDockerContainerUpdate,
		Delete:        resourceDockerContainerDelete,
		MigrateState:  resourceDockerContainerMigrateState,
		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rm": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"start": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"attach": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"logs": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			// Indicates whether the container must be running.
			//
			// An assumption is made that configured containers
			// should be running; if not, they should not be in
			// the configuration. Therefore a stopped container
			// should be started. Set to false to have the
			// provider leave the container alone.
			//
			// Actively-debugged containers are likely to be
			// stopped and started manually, and Docker has
			// some provisions for restarting containers that
			// stop. The utility here comes from the fact that
			// this will delete and re-create the container
			// following the principle that the containers
			// should be pristine when started.
			"must_run": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"exit_code": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"container_logs": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domainname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_opts": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_search": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"publish_all_ports": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"restart": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "no",
				ValidateFunc: validateStringMatchesPattern(`^(no|on-failure|always|unless-stopped)$`),
			},

			"max_retry_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"capabilities": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"drop": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"volumes": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_container": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"container_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"host_path": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateDockerContainerPath,
						},

						"volume_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"read_only": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"ports": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"external": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"ip": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "0.0.0.0",
							Optional: true,
							ForceNew: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
				DiffSuppressFunc: suppressIfPortsDidNotChangeForMigrationV0ToV1(),
			},

			"host": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"ulimit": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"soft": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"hard": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"env": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"links": &schema.Schema{
				Type:       schema.TypeSet,
				Optional:   true,
				ForceNew:   true,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Set:        schema.HashString,
				Deprecated: "The --link flag is a legacy feature of Docker. It may eventually be removed.",
			},

			"ip_address": &schema.Schema{
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"ip_prefix_length": &schema.Schema{
				Type:       schema.TypeInt,
				Computed:   true,
				Deprecated: "Use ip_prefix_length from ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"gateway": &schema.Schema{
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use gateway from ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"bridge": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_data": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"gateway": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"privileged": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"devices": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"container_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"permissions": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"destroy_grace_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"memory": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(0),
			},

			"memory_swap": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(-1),
			},

			"cpu_shares": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(0),
			},

			"cpu_set": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateStringMatchesPattern(`^\d+([,-]\d+)*$`),
			},

			"log_driver": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "json-file",
				ValidateFunc: validateStringMatchesPattern(`^(json-file|syslog|journald|gelf|fluentd|awslogs)$`),
			},

			"log_opts": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_alias": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Description: "Set an alias for the container in all specified networks",
				Deprecated:  "Use networks_advanced instead. Will be removed in v2.0.0",
			},

			"network_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"networks": &schema.Schema{
				Type:       schema.TypeSet,
				Optional:   true,
				ForceNew:   true,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Set:        schema.HashString,
				Deprecated: "Use networks_advanced instead. Will be removed in v2.0.0",
			},

			"networks_advanced": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"aliases": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"ipv4_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"ipv6_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"pid_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"userns_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"upload": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							// This is intentional. The container is mutated once, and never updated later.
							// New configuration forces a new deployment, even with the same binaries.
							ForceNew: true,
						},
						"file": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"executable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},
					},
				},
			},

			"healthcheck": &schema.Schema{
				Type:        schema.TypeList,
				Description: "A test to perform to check that the container is healthy",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": &schema.Schema{
							Type:        schema.TypeList,
							Description: "The test to perform as list",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Time between running the check (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Maximum time to allow one check to run (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"start_period": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"retries": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Consecutive failures needed to report unhealthy",
							Optional:     true,
							Default:      0,
							ValidateFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},
		},
	}
}

func suppressIfPortsDidNotChangeForMigrationV0ToV1() schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		portsOldRaw, portsNewRaw := d.GetChange("ports")
		portsOld := portsOldRaw.([]interface{})
		portsNew := portsNewRaw.([]interface{})
		if len(portsOld) != len(portsNew) {
			log.Printf("[DEBUG] suppress diff ports: old and new don't have the same length")
			return false
		}
		log.Printf("[DEBUG] suppress diff ports: old and new have same length")

		for _, portOld := range portsOld {
			portOldMapped := portOld.(map[string]interface{})
			oldInternalPort := portOldMapped["internal"]
			portFound := false
			for _, portNew := range portsNew {
				portNewMapped := portNew.(map[string]interface{})
				newInternalPort := portNewMapped["internal"]
				// port is still there in new
				if newInternalPort == oldInternalPort {
					log.Printf("[DEBUG] suppress diff ports: comparing port '%v'", oldInternalPort)
					portFound = true
					if portNewMapped["external"] != portOldMapped["external"] {
						log.Printf("[DEBUG] suppress diff ports: 'external' changed for '%v'", oldInternalPort)
						return false
					}
					if portNewMapped["ip"] != portOldMapped["ip"] {
						log.Printf("[DEBUG] suppress diff ports: 'ip' changed for '%v'", oldInternalPort)
						return false
					}
					if portNewMapped["protocol"] != portOldMapped["protocol"] {
						log.Printf("[DEBUG] suppress diff ports: 'protocol' changed for '%v'", oldInternalPort)
						return false
					}
				}
			}
			// port was deleted or exchanges in new
			if !portFound {
				log.Printf("[DEBUG] suppress diff ports: port was deleted '%v'", oldInternalPort)
				return false
			}
		}
		return true
	}
}
