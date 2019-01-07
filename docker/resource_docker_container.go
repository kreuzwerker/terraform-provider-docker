package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerContainerCreate,
		Read:   resourceDockerContainerRead,
		Update: resourceDockerContainerUpdate,
		Delete: resourceDockerContainerDelete,

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
			},

			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "no",
				ValidateFunc: validateStringMatchesPattern(`^(no|on-failure|always|unless-stopped)$`),
			},

			"max_retry_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"capabilities": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
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
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateDockerContainerPath,
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
						},

						"protocol": {
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
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
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"memory": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(0),
			},

			"memory_swap": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(-1),
			},

			"cpu_shares": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerGeqThan(0),
			},

			"cpu_set": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateStringMatchesPattern(`^\d+([,-]\d+)*$`),
			},

			"log_driver": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "json-file",
				ValidateFunc: validateStringMatchesPattern(`^(json-file|syslog|journald|gelf|fluentd|awslogs)$`),
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
							Required: true,
							// This is intentional. The container is mutated once, and never updated later.
							// New configuration forces a new deployment, even with the same binaries.
							ForceNew: true,
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
					},
				},
			},

			"healthcheck": {
				Type:        schema.TypeList,
				Description: "A test to perform to check that the container is healthy",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": {
							Type:        schema.TypeList,
							Description: "The test to perform as list",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": {
							Type:         schema.TypeString,
							Description:  "Time between running the check (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:         schema.TypeString,
							Description:  "Maximum time to allow one check to run (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"start_period": {
							Type:         schema.TypeString,
							Description:  "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"retries": {
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
