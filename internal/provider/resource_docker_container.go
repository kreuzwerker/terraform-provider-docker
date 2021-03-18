package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDockerContainerCreate,
		ReadContext:   resourceDockerContainerRead,
		UpdateContext: resourceDockerContainerUpdate,
		DeleteContext: resourceDockerContainerDelete,
		MigrateState:  resourceDockerContainerMigrateState,
		SchemaVersion: 2,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 1,
				Type:    resourceDockerContainerV1().CoreConfigSchema().ImpliedType(),
				Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					// TODO do the ohter V0-to-V1 migration, unless we're okay
					// with breaking for users who straggled on their docker
					// provider version

					return migrateContainerLabels(rawState), nil
				},
			},
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rm": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"read_only": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: true,
			},

			"start": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"attach": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"logs": {
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
			"must_run": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"exit_code": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"container_logs": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// DiffSuppressFunc: suppressIfSHAwasAdded(), // TODO mvogel
			},

			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"domainname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"command": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as a no-op, which is Docker's default value
					if newV == "" {
						newV = oldV
					}
					return oldV == newV
				},
			},

			"dns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_opts": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_search": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"publish_all_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"restart": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "no",
				ValidateDiagFunc: validateStringMatchesPattern(`^(no|on-failure|always|unless-stopped)$`),
			},

			"max_retry_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"working_dir": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as a no-op, which is Docker's default behavior
					if newV == "" {
						newV = oldV
					}
					return oldV == newV
				},
			},
			"remove_volumes": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"capabilities": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				// TODO implement DiffSuppressFunc
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"drop": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
			"security_opts": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of string values to customize labels for MLS systems, such as SELinux. See https://docs.docker.com/engine/reference/run/#security-configuration",
				Set:         schema.HashString,
			},
			"mounts": {
				Type:        schema.TypeSet,
				Description: "Specification for mounts to be added to containers created as part of the service",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": {
							Type:        schema.TypeString,
							Description: "Container path",
							Required:    true,
						},
						"source": {
							Type:        schema.TypeString,
							Description: "Mount source (e.g. a volume name, a host path)",
							Optional:    true,
						},
						"type": {
							Type:             schema.TypeString,
							Description:      "The mount type",
							Required:         true,
							ValidateDiagFunc: validateStringMatchesPattern(`^(bind|volume|tmpfs)$`),
						},
						"read_only": {
							Type:        schema.TypeBool,
							Description: "Whether the mount should be read-only",
							Optional:    true,
						},
						"bind_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the bind type",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"propagation": {
										Type:             schema.TypeString,
										Description:      "A propagation mode with the value",
										Optional:         true,
										ValidateDiagFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
									},
								},
							},
						},
						"volume_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the volume type",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"no_copy": {
										Type:        schema.TypeBool,
										Description: "Populate volume with data from the target",
										Optional:    true,
									},
									"labels": {
										Type:        schema.TypeSet,
										Description: "User-defined key/value metadata",
										Optional:    true,
										Elem:        labelSchema,
									},
									"driver_name": {
										Type:        schema.TypeString,
										Description: "Name of the driver to use to create the volume",
										Optional:    true,
									},
									"driver_options": {
										Type:        schema.TypeMap,
										Description: "key/value map of driver specific options",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"tmpfs_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the tmpfs type",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"size_bytes": {
										Type:        schema.TypeInt,
										Description: "The size for the tmpfs mount in bytes",
										Optional:    true,
									},
									"mode": {
										Type:        schema.TypeInt,
										Description: "The permission mode for the tmpfs mount in an integer",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
			"volumes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_container": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"host_path": {
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDockerContainerPath(),
						},

						"volume_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"read_only": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"tmpfs": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"ports": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"external": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"ip": {
							Type:     schema.TypeString,
							Default:  "0.0.0.0",
							Optional: true,
							ForceNew: true,
							StateFunc: func(val interface{}) string {
								// Empty IP assignments default to 0.0.0.0
								if val.(string) == "" {
									return "0.0.0.0"
								}

								return val.(string)
							},
						},

						"protocol": {
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
				DiffSuppressFunc: suppressIfPortsDidNotChangeForMigrationV0ToV1(),
			},

			"host": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"host": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"ulimit": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"soft": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"hard": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"links": {
				Type:       schema.TypeSet,
				Optional:   true,
				ForceNew:   true,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Set:        schema.HashString,
				Deprecated: "The --link flag is a legacy feature of Docker. It may eventually be removed.",
			},

			"ip_address": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"ip_prefix_length": {
				Type:       schema.TypeInt,
				Computed:   true,
				Deprecated: "Use ip_prefix_length from ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"gateway": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use gateway from ip_adresses_data instead. This field exposes the data of the container's first network.",
			},

			"bridge": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_data": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_prefix_length": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"gateway": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"global_ipv6_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"global_ipv6_prefix_length": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ipv6_gateway": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"privileged": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"devices": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_path": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"permissions": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"destroy_grace_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"labels": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     labelSchema,
			},

			"memory": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"memory_swap": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(-1),
			},

			"shm_size": {
				Type:             schema.TypeInt,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"cpu_shares": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"cpu_set": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringMatchesPattern(`^\d+([,-]\d+)*$`),
			},

			"log_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "json-file",
			},

			"log_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_alias": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Description: "Set an alias for the container in all specified networks",
				Deprecated:  "Use networks_advanced instead. Will be removed in v2.0.0",
			},

			"network_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as "default", which is Docker's default value
					if oldV == "" {
						oldV = "default"
					}
					if newV == "" {
						newV = "default"
					}
					return oldV == newV
				},
			},

			"networks": {
				Type:       schema.TypeSet,
				Optional:   true,
				ForceNew:   true,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Set:        schema.HashString,
				Deprecated: "Use networks_advanced instead. Will be removed in v2.0.0",
			},

			"networks_advanced": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"aliases": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"ipv4_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"ipv6_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"pid_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"userns_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"upload": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": {
							Type:     schema.TypeString,
							Optional: true,
							// This is intentional. The container is mutated once, and never updated later.
							// New configuration forces a new deployment, even with the same binaries.
							ForceNew: true,
						},
						"content_base64": {
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateStringIsBase64Encoded(),
						},
						"file": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"executable": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},
						"source": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"source_hash": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"healthcheck": {
				Type:        schema.TypeList,
				Description: "A test to perform to check that the container is healthy",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": {
							Type:        schema.TypeList,
							Description: "The test to perform as list",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": {
							Type:             schema.TypeString,
							Description:      "Time between running the check (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:             schema.TypeString,
							Description:      "Maximum time to allow one check to run (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"start_period": {
							Type:             schema.TypeString,
							Description:      "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"retries": {
							Type:             schema.TypeInt,
							Description:      "Consecutive failures needed to report unhealthy",
							Optional:         true,
							Default:          0,
							ValidateDiagFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},

			"sysctls": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"ipc_mode": {
				Type:        schema.TypeString,
				Description: "IPC sharing mode for the container",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			"group_add": {
				Type:        schema.TypeSet,
				Description: "Additional groups for the container user",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"init": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"tty": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"stdin_open": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
		},
	}
}

func suppressIfPortsDidNotChangeForMigrationV0ToV1() schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		if k == "ports.#" && old != new {
			log.Printf("[DEBUG] suppress diff ports: old and new don't have the same length")
			return false
		}
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
					if portNewMapped["protocol"] != portOldMapped["protocol"] {
						if containsPortWithProtocol(portsNew, portOldMapped["internal"], portOldMapped["protocol"]) {
							log.Printf("[DEBUG] suppress diff ports: found another port in new list with the same protocol for '%v", oldInternalPort)
							continue
						}

						log.Printf("[DEBUG] suppress diff ports: 'protocol' changed for '%v'", oldInternalPort)
						return false
					}
					if portNewMapped["external"] != portOldMapped["external"] {
						log.Printf("[DEBUG] suppress diff ports: 'external' changed for '%v'", oldInternalPort)
						return false
					}
					if portNewMapped["ip"] != portOldMapped["ip"] {
						log.Printf("[DEBUG] suppress diff ports: 'ip' changed for '%v'", oldInternalPort)
						return false
					}

					portFound = true
					break
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

func containsPortWithProtocol(ports []interface{}, searchInternalPort, searchProtocol interface{}) bool {
	for _, port := range ports {
		portMapped := port.(map[string]interface{})
		internalPort := portMapped["internal"]
		protocol := portMapped["protocol"]
		if internalPort == searchInternalPort && protocol == searchProtocol {
			return true
		}
	}

	return false
}
