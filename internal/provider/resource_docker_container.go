package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the lifecycle of a Docker container.",

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
				Type:        schema.TypeString,
				Description: "The name of the container.",
				Required:    true,
				ForceNew:    true,
			},

			"rm": {
				Type:        schema.TypeBool,
				Description: "If `true`, then the container will be automatically removed after his execution. Terraform won't check this container after creation. Defaults to `false`.",
				Default:     false,
				Optional:    true,
			},

			"read_only": {
				Type:        schema.TypeBool,
				Description: "If `true`, the container will be started as readonly. Defaults to `false`.",
				Default:     false,
				Optional:    true,
				ForceNew:    true,
			},

			"start": {
				Type:        schema.TypeBool,
				Description: "If `true`, then the Docker container will be started after creation. If `false`, then the container is only created. Defaults to `true`.",
				Default:     true,
				Optional:    true,
			},

			"attach": {
				Type:        schema.TypeBool,
				Description: "If `true` attach to the container after its creation and waits the end of its execution. Defaults to `false`.",
				Default:     false,
				Optional:    true,
			},

			"logs": {
				Type:        schema.TypeBool,
				Description: "Save the container logs (`attach` must be enabled). Defaults to `false`.",
				Default:     false,
				Optional:    true,
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
				Type:        schema.TypeBool,
				Description: "If `true`, then the Docker container will be kept running. If `false`, then as long as the container exists, Terraform assumes it is successful. Defaults to `true`.",
				Default:     true,
				Optional:    true,
			},

			"exit_code": {
				Type:        schema.TypeInt,
				Description: "The exit code of the container if its execution is done (`must_run` must be disabled).",
				Computed:    true,
			},

			"container_logs": {
				Type:        schema.TypeString,
				Description: "The logs of the container if its execution is done (`attach` must be disabled).",
				Computed:    true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": {
				Type:        schema.TypeString,
				Description: "The ID of the image to back this container. The easiest way to get this value is to use the `docker_image` resource as is shown in the example.",
				Required:    true,
				ForceNew:    true,
			},

			"hostname": {
				Type:        schema.TypeString,
				Description: "Hostname of the container.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},

			"domainname": {
				Type:        schema.TypeString,
				Description: "Domain name of the container.",
				Optional:    true,
				ForceNew:    true,
			},

			"command": {
				Type:        schema.TypeList,
				Description: "The command to use to start the container. For example, to run `/usr/bin/myprogram -f baz.conf` set the command to be `[\"/usr/bin/myprogram\",\"-\",\"baz.con\"]`.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": {
				Type:        schema.TypeList,
				Description: "The command to use as the Entrypoint for the container. The Entrypoint allows you to configure a container to run as an executable. For example, to run `/usr/bin/myprogram` when starting a container, set the entrypoint to be `\"/usr/bin/myprogra\"]`.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"user": {
				Type:        schema.TypeString,
				Description: "User used for run the first process. Format is `user` or `user:group` which user and group can be passed literraly or by name.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as a no-op, which is Docker's default value
					if newV == "" {
						newV = oldV
					}
					return oldV == newV
				},
			},

			"dns": {
				Type:        schema.TypeSet,
				Description: "DNS servers to use.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"dns_opts": {
				Type:        schema.TypeSet,
				Description: "DNS options used by the DNS provider(s), see `resolv.conf` documentation for valid list of options.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"dns_search": {
				Type:        schema.TypeSet,
				Description: "DNS search domains that are used when bare unqualified hostnames are used inside of the container.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"publish_all_ports": {
				Type:        schema.TypeBool,
				Description: "Publish all ports of the container.",
				Optional:    true,
				ForceNew:    true,
			},

			"restart": {
				Type:             schema.TypeString,
				Description:      "The restart policy for the container. Must be one of 'no', 'on-failure', 'always', 'unless-stopped'. Defaults to `no`.",
				Default:          "no",
				Optional:         true,
				ValidateDiagFunc: validateStringMatchesPattern(`^(no|on-failure|always|unless-stopped)$`),
			},

			"max_retry_count": {
				Type:        schema.TypeInt,
				Description: "The maximum amount of times to an attempt a restart when `restart` is set to 'on-failure'.",
				Optional:    true,
			},
			"working_dir": {
				Type:        schema.TypeString,
				Description: "The working directory for commands to run in.",
				Optional:    true,
				ForceNew:    true,
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as a no-op, which is Docker's default behavior
					if newV == "" {
						newV = oldV
					}
					return oldV == newV
				},
			},
			"remove_volumes": {
				Type:        schema.TypeBool,
				Description: "If `true`, it will remove anonymous volumes associated with the container. Defaults to `true`.",
				Default:     true,
				Optional:    true,
			},
			"capabilities": {
				Type:        schema.TypeSet,
				Description: "Add or drop certrain linux capabilities.",
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				// TODO implement DiffSuppressFunc
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:        schema.TypeSet,
							Description: "List of linux capabilities to add.",
							Optional:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},

						"drop": {
							Type:        schema.TypeSet,
							Description: "List of linux capabilities to drop.",
							Optional:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
					},
				},
			},
			"security_opts": {
				Type:        schema.TypeSet,
				Description: "List of string values to customize labels for MLS systems, such as SELinux. See https://docs.docker.com/engine/reference/run/#security-configuration.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"mounts": {
				Type:        schema.TypeSet,
				Description: "Specification for mounts to be added to containers created as part of the service.",
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
							Description: "Mount source (e.g. a volume name, a host path).",
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
							Description: "Whether the mount should be read-only.",
							Optional:    true,
						},
						"bind_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the bind type.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"propagation": {
										Type:             schema.TypeString,
										Description:      "A propagation mode with the value.",
										Optional:         true,
										ValidateDiagFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
									},
								},
							},
						},
						"volume_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the volume type.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"no_copy": {
										Type:        schema.TypeBool,
										Description: "Populate volume with data from the target.",
										Optional:    true,
									},
									"labels": {
										Type:        schema.TypeSet,
										Description: "User-defined key/value metadata.",
										Optional:    true,
										Elem:        labelSchema,
									},
									"driver_name": {
										Type:        schema.TypeString,
										Description: "Name of the driver to use to create the volume.",
										Optional:    true,
									},
									"driver_options": {
										Type:        schema.TypeMap,
										Description: "key/value map of driver specific options.",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"tmpfs_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the tmpfs type.",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"size_bytes": {
										Type:        schema.TypeInt,
										Description: "The size for the tmpfs mount in bytes.",
										Optional:    true,
									},
									"mode": {
										Type:        schema.TypeInt,
										Description: "The permission mode for the tmpfs mount in an integer.",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
			"volumes": {
				Type:        schema.TypeSet,
				Description: "Spec for mounting volumes in the container.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_container": {
							Type:        schema.TypeString,
							Description: "The container where the volume is coming from.",
							Optional:    true,
							ForceNew:    true,
						},
						"container_path": {
							Type:        schema.TypeString,
							Description: "The path in the container where the volume will be mounted.",
							Optional:    true,
							ForceNew:    true,
						},
						"host_path": {
							Type:             schema.TypeString,
							Description:      "The path on the host where the volume is coming from.",
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDockerContainerPath(),
						},
						"volume_name": {
							Type:        schema.TypeString,
							Description: "The name of the docker volume which should be mounted.",
							Optional:    true,
							ForceNew:    true,
						},
						"read_only": {
							Type:        schema.TypeBool,
							Description: "If `true`, this volume will be readonly. Defaults to `false`.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"tmpfs": {
				Type:        schema.TypeMap,
				Description: "A map of container directories which should be replaced by `tmpfs mounts`, and their corresponding mount options.",
				Optional:    true,
			},
			"ports": {
				Type:        schema.TypeList,
				Description: "Publish a container's port(s) to the host.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": {
							Type:        schema.TypeInt,
							Description: "Port within the container.",
							Required:    true,
							ForceNew:    true,
						},

						"external": {
							Type:        schema.TypeInt,
							Description: "Port exposed out of the container. If not given a free random port `>= 32768` will be used.",
							Optional:    true,
							Computed:    true,
							ForceNew:    true,
						},

						"ip": {
							Type:        schema.TypeString,
							Description: "IP address/mask that can access this port. Defaults to `0.0.0.0`.",
							Default:     "0.0.0.0",
							Optional:    true,
							ForceNew:    true,
							StateFunc: func(val interface{}) string {
								// Empty IP assignments default to 0.0.0.0
								if val.(string) == "" {
									return "0.0.0.0"
								}

								return val.(string)
							},
						},

						"protocol": {
							Type:        schema.TypeString,
							Description: "Protocol that can be used over this port. Defaults to `tcp`.",
							Default:     "tcp",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
				DiffSuppressFunc: suppressIfPortsDidNotChangeForMigrationV0ToV1(),
			},

			"host": {
				Type:        schema.TypeSet,
				Description: "Additional hosts to add to the container.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:        schema.TypeString,
							Description: "IP address this hostname should resolve to.",
							Required:    true,
							ForceNew:    true,
						},

						"host": {
							Type:        schema.TypeString,
							Description: "Hostname to add",
							Required:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"ulimit": {
				Type:        schema.TypeSet,
				Description: "Ulimit options to add.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the ulimit",
							Required:    true,
							ForceNew:    true,
						},
						"soft": {
							Type:        schema.TypeInt,
							Description: "The soft limit",
							Required:    true,
							ForceNew:    true,
						},
						"hard": {
							Type:        schema.TypeInt,
							Description: "The hard limit",
							Required:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"env": {
				Type:        schema.TypeSet,
				Description: "Environment variables to set in the form of `KEY=VALUE`, e.g. `DEBUG=0`",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"links": {
				Type:        schema.TypeSet,
				Description: "Set of links for link based connectivity between containers that are running on the same host.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Deprecated:  "The --link flag is a legacy feature of Docker. It may eventually be removed.",
			},

			"ip_address": {
				Type:        schema.TypeString,
				Description: "The IP address of the container.",
				Computed:    true,
				Deprecated:  "Use `network_data` instead. The IP address of the container's first network it.",
			},

			"ip_prefix_length": {
				Type:        schema.TypeInt,
				Description: "The IP prefix length of the container.",
				Computed:    true,
				Deprecated:  "Use `network_data` instead. The IP prefix length of the container as read from its NetworkSettings.",
			},

			"gateway": {
				Type:        schema.TypeString,
				Description: "The network gateway of the container.",
				Computed:    true,
				Deprecated:  "Use `network_data` instead. The network gateway of the container as read from its NetworkSettings.",
			},

			"bridge": {
				Type:        schema.TypeString,
				Description: "The network bridge of the container as read from its NetworkSettings.",
				Computed:    true,
			},

			"network_data": {
				Type:        schema.TypeList,
				Description: "The data of the networks the container is connected to.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_name": {
							Type:        schema.TypeString,
							Description: "The name of the network",
							Computed:    true,
						},
						"ip_address": {
							Type:        schema.TypeString,
							Description: "The IP address of the container.",
							Computed:    true,
							Deprecated:  "Use `network_data` instead. The IP address of the container's first network it.",
						},
						"ip_prefix_length": {
							Type:        schema.TypeInt,
							Description: "The IP prefix length of the container.",
							Computed:    true,
							Deprecated:  "Use `network_data` instead. The IP prefix length of the container as read from its NetworkSettings.",
						},
						"gateway": {
							Type:        schema.TypeString,
							Description: "The network gateway of the container.",
							Computed:    true,
							Deprecated:  "Use `network_data` instead. The network gateway of the container as read from its NetworkSettings.",
						},
						"global_ipv6_address": {
							Type:        schema.TypeString,
							Description: "The IPV6 address of the container.",
							Computed:    true,
						},
						"global_ipv6_prefix_length": {
							Type:        schema.TypeInt,
							Description: "The IPV6 prefix length address of the container.",
							Computed:    true,
						},
						"ipv6_gateway": {
							Type:        schema.TypeString,
							Description: "The IPV6 gateway of the container.",
							Computed:    true,
						},
					},
				},
			},

			"privileged": {
				Type:        schema.TypeBool,
				Description: "If `true`, the container runs in privileged mode.",
				Optional:    true,
				ForceNew:    true,
			},

			"devices": {
				Type:        schema.TypeSet,
				Description: "Bind devices to the container.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_path": {
							Type:        schema.TypeString,
							Description: "The path on the host where the device is located.",
							Required:    true,
							ForceNew:    true,
						},
						"container_path": {
							Type:        schema.TypeString,
							Description: "The path in the container where the device will be bound.",
							Optional:    true,
							ForceNew:    true,
						},
						"permissions": {
							Type:        schema.TypeString,
							Description: "The cgroup permissions given to the container to access the device. Defaults to `rwm`.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"destroy_grace_seconds": {
				Type:        schema.TypeInt,
				Description: "If defined will attempt to stop the container before destroying. Container will be destroyed after `n` seconds or on successful stop.",
				Optional:    true,
			},

			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Elem:        labelSchema,
			},

			"memory": {
				Type:             schema.TypeInt,
				Description:      "The memory limit for the container in MBs.",
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"memory_swap": {
				Type:             schema.TypeInt,
				Description:      "The total memory limit (memory + swap) for the container in MBs. This setting may compute to `-1` after `terraform apply` if the target host doesn't support memory swap, when that is the case docker will use a soft limitation.",
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(-1),
			},

			"shm_size": {
				Type:             schema.TypeInt,
				Description:      "Size of `/dev/shm` in MBs.",
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"cpu_shares": {
				Type:             schema.TypeInt,
				Description:      "CPU shares (relative weight) for the container.",
				Optional:         true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"cpu_set": {
				Type:             schema.TypeString,
				Description:      "A comma-separated list or hyphen-separated range of CPUs a container can use, e.g. `0-1`.",
				Optional:         true,
				ValidateDiagFunc: validateStringMatchesPattern(`^\d+([,-]\d+)*$`),
			},

			"log_driver": {
				Type:        schema.TypeString,
				Description: "The logging driver to use for the container. Defaults to `json-file`.",
				Default:     "json-file",
				Optional:    true,
				ForceNew:    true,
			},

			"log_opts": {
				Type:        schema.TypeMap,
				Description: "Key/value pairs to use as options for the logging driver.",
				Optional:    true,
				ForceNew:    true,
			},

			"network_alias": {
				Type:        schema.TypeSet,
				Description: "Set an alias for the container in all specified networks",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Deprecated:  "Use networks_advanced instead. Will be removed in v3.0.0",
			},

			"network_mode": {
				Type:        schema.TypeString,
				Description: "Network mode of the container.",
				Optional:    true,
				ForceNew:    true,
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
				Type:        schema.TypeSet,
				Description: "ID of the networks in which the container is.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Deprecated:  "Use networks_advanced instead. Will be removed in v3.0.0",
			},

			"networks_advanced": {
				Type:        schema.TypeSet,
				Description: "The networks the container is attached to",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the network.",
							Required:    true,
							ForceNew:    true,
						},
						"aliases": {
							Type:        schema.TypeSet,
							Description: "The network aliases of the container in the specific network.",
							Optional:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"ipv4_address": {
							Type:        schema.TypeString,
							Description: "The IPV4 address of the container in the specific network.",
							Optional:    true,
							ForceNew:    true,
						},
						"ipv6_address": {
							Type:        schema.TypeString,
							Description: "The IPV6 address of the container in the specific network.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"pid_mode": {
				Type:        schema.TypeString,
				Description: "he PID (Process) Namespace mode for the container. Either `container:<name|id>` or `host`.",
				Optional:    true,
				ForceNew:    true,
			},
			"userns_mode": {
				Type:        schema.TypeString,
				Description: "Sets the usernamespace mode for the container when usernamespace remapping option is enabled.",
				Optional:    true,
				ForceNew:    true,
			},

			"upload": {
				Type:        schema.TypeSet,
				Description: "Specifies files to upload to the container before starting it. Only one of `content` or `content_base64` can be set and at least one of them has to be set.",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": {
							Type:        schema.TypeString,
							Description: "Literal string value to use as the object content, which will be uploaded as UTF-8-encoded text. Conflicts with `content_base64` & `source`",
							Optional:    true,
							// This is intentional. The container is mutated once, and never updated later.
							// New configuration forces a new deployment, even with the same binaries.
							ForceNew: true,
						},
						"content_base64": {
							Type:             schema.TypeString,
							Description:      "Base64-encoded data that will be decoded and uploaded as raw bytes for the object content. This allows safely uploading non-UTF8 binary data, but is recommended only for larger binary content such as the result of the `base64encode` interpolation function. See [here](https://github.com/terraform-providers/terraform-provider-docker/issues/48#issuecomment-374174588) for the reason. Conflicts with `content` & `source`",
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateStringIsBase64Encoded(),
						},
						"file": {
							Type:        schema.TypeString,
							Description: "Path to the file in the container where is upload goes to",
							Required:    true,
							ForceNew:    true,
						},
						"executable": {
							Type:        schema.TypeBool,
							Description: "If `true`, the file will be uploaded with user executable permission. Defaults to `false`.",
							Default:     false,
							Optional:    true,
							ForceNew:    true,
						},
						"source": {
							Type:        schema.TypeString,
							Description: "A filename that references a file which will be uploaded as the object content. This allows for large file uploads that do not get stored in state. Conflicts with `content` & `content_base64`",
							Optional:    true,
							ForceNew:    true,
						},
						"source_hash": {
							Type:        schema.TypeString,
							Description: "If using `source`, this will force an update if the file content has updated but the filename has not. ",
							Optional:    true,
							ForceNew:    true,
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
							Description: "Command to run to check health. For example, to run `curl -f localhost/health` set the command to be `[\"CMD\", \"curl\", \"-f\", \"localhost/health\"]`.",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": {
							Type:             schema.TypeString,
							Description:      "Time between running the check (ms|s|m|h). Defaults to `0s`.",
							Default:          "0s",
							Optional:         true,
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:             schema.TypeString,
							Description:      "Maximum time to allow one check to run (ms|s|m|h). Defaults to `0s`.",
							Default:          "0s",
							Optional:         true,
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"start_period": {
							Type:             schema.TypeString,
							Description:      "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h). Defaults to `0s`.",
							Default:          "0s",
							Optional:         true,
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"retries": {
							Type:             schema.TypeInt,
							Description:      "Consecutive failures needed to report unhealthy. Defaults to `0`.",
							Default:          0,
							Optional:         true,
							ValidateDiagFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},

			"sysctls": {
				Type:        schema.TypeMap,
				Description: "A map of kernel parameters (sysctls) to set in the container.",
				Optional:    true,
				ForceNew:    true,
			},
			"ipc_mode": {
				Type:        schema.TypeString,
				Description: "IPC sharing mode for the container. Possible values are: `none`, `private`, `shareable`, `container:<name|id>` or `host`.",
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
				Type:        schema.TypeBool,
				Description: "Configured whether an init process should be injected for this container. If unset this will default to the `dockerd` defaults.",
				Optional:    true,
				Computed:    true,
			},
			"tty": {
				Type:        schema.TypeBool,
				Description: "If `true`, allocate a pseudo-tty (`docker run -t`). Defaults to `false`.",
				Default:     false,
				Optional:    true,
				ForceNew:    true,
			},
			"stdin_open": {
				Type:        schema.TypeBool,
				Description: "If `true`, keep STDIN open even if not attached (`docker run -i`). Defaults to `false`.",
				Default:     false,
				Optional:    true,
				ForceNew:    true,
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
