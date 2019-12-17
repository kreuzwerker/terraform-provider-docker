package docker

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// resourceDockerService create a docker service
// https://docs.docker.com/engine/api/v1.32/#operation/ServiceCreate
func resourceDockerService() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerServiceCreate,
		Read:   resourceDockerServiceRead,
		Update: resourceDockerServiceUpdate,
		Delete: resourceDockerServiceDelete,
		Exists: resourceDockerServiceExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"auth": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server_address": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"username": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
						},
						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Sensitive:   true,
						},
					},
				},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the service",
				Required:    true,
				ForceNew:    true,
			},
			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				Computed:    true,
				Elem:        labelSchema,
			},
			"task_spec": {
				Type:        schema.TypeList,
				Description: "User modifiable task configuration",
				MaxItems:    1,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_spec": {
							Type:        schema.TypeList,
							Description: "The spec for each container",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"image": {
										Type:             schema.TypeString,
										Description:      "The image name to use for the containers of the service",
										Required:         true,
										DiffSuppressFunc: suppressIfSHAwasAdded(),
									},
									"labels": {
										Type:        schema.TypeSet,
										Description: "User-defined key/value metadata",
										Optional:    true,
										Elem:        labelSchema,
									},
									"command": {
										Type:        schema.TypeList,
										Description: "The command to be run in the image",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"args": {
										Type:        schema.TypeList,
										Description: "Arguments to the command",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "The hostname to use for the container, as a valid RFC 1123 hostname",
										Optional:    true,
									},
									"env": {
										Type:        schema.TypeMap,
										Description: "A list of environment variables in the form VAR=\"value\"",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"dir": {
										Type:        schema.TypeString,
										Description: "The working directory for commands to run in",
										Optional:    true,
									},
									"user": {
										Type:        schema.TypeString,
										Description: "The user inside the container",
										Optional:    true,
									},
									"groups": {
										Type:        schema.TypeList,
										Description: "A list of additional groups that the container process will run as",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"privileges": {
										Type:        schema.TypeList,
										Description: "Security options for the container",
										MaxItems:    1,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"credential_spec": {
													Type:        schema.TypeList,
													Description: "CredentialSpec for managed service account (Windows only)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"file": {
																Type:        schema.TypeString,
																Description: "Load credential spec from this file",
																Optional:    true,
															},
															"registry": {
																Type:        schema.TypeString,
																Description: "Load credential spec from this value in the Windows registry",
																Optional:    true,
															},
														},
													},
												},
												"se_linux_context": {
													Type:        schema.TypeList,
													Description: "SELinux labels of the container",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"disable": {
																Type:        schema.TypeBool,
																Description: "Disable SELinux",
																Optional:    true,
															},
															"user": {
																Type:        schema.TypeString,
																Description: "SELinux user label",
																Optional:    true,
															},
															"role": {
																Type:        schema.TypeString,
																Description: "SELinux role label",
																Optional:    true,
															},
															"type": {
																Type:        schema.TypeString,
																Description: "SELinux type label",
																Optional:    true,
															},
															"level": {
																Type:        schema.TypeString,
																Description: "SELinux level label",
																Optional:    true,
															},
														},
													},
												},
											},
										},
									},
									"read_only": {
										Type:        schema.TypeBool,
										Description: "Mount the container's root filesystem as read only",
										Optional:    true,
									},
									"mounts": {
										Type:        schema.TypeSet,
										Description: "Specification for mounts to be added to containers created as part of the service",
										Optional:    true,
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
													Type:         schema.TypeString,
													Description:  "The mount type",
													Required:     true,
													ValidateFunc: validateStringMatchesPattern(`^(bind|volume|tmpfs)$`),
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
																Type:         schema.TypeString,
																Description:  "A propagation mode with the value",
																Optional:     true,
																ValidateFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
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
																Description: "Name of the driver to use to create the volume.",
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
									"stop_signal": {
										Type:        schema.TypeString,
										Description: "Signal to stop the container",
										Optional:    true,
									},
									"stop_grace_period": {
										Type:         schema.TypeString,
										Description:  "Amount of time to wait for the container to terminate before forcefully removing it (ms|s|m|h)",
										Optional:     true,
										Computed:     true,
										ValidateFunc: validateDurationGeq0(),
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
									"hosts": {
										Type:        schema.TypeSet,
										Description: "A list of hostname/IP mappings to add to the container's hosts file.",
										Optional:    true,
										ForceNew:    true,
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
									"dns_config": {
										Type:        schema.TypeList,
										Description: "Specification for DNS related configurations in resolver configuration file (resolv.conf)",
										MaxItems:    1,
										Optional:    true,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nameservers": {
													Type:        schema.TypeList,
													Description: "The IP addresses of the name servers",
													Required:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"search": {
													Type:        schema.TypeList,
													Description: "A search list for host-name lookup",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"options": {
													Type:        schema.TypeList,
													Description: "A list of internal resolver variables to be modified (e.g., debug, ndots:3, etc.)",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"secrets": {
										Type:        schema.TypeSet,
										Description: "References to zero or more secrets that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"secret_id": {
													Type:        schema.TypeString,
													Description: "ID of the specific secret that we're referencing",
													Required:    true,
												},
												"secret_name": {
													Type:        schema.TypeString,
													Description: "Name of the secret that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": {
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"configs": {
										Type:        schema.TypeSet,
										Description: "References to zero or more configs that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"config_id": {
													Type:        schema.TypeString,
													Description: "ID of the specific config that we're referencing",
													Required:    true,
												},
												"config_name": {
													Type:        schema.TypeString,
													Description: "Name of the config that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": {
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"isolation": {
										Type:         schema.TypeString,
										Description:  "Isolation technology of the containers running the service. (Windows only)",
										Optional:     true,
										Default:      "default",
										ValidateFunc: validateStringMatchesPattern(`^(default|process|hyperv)$`),
									},
								},
							},
						},
						"resources": {
							Type:        schema.TypeList,
							Description: "Resource requirements which apply to each individual container created as part of the service",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"limits": {
										Type:        schema.TypeList,
										Description: "Describes the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": {
													Type:        schema.TypeInt,
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Optional:    true,
												},
												"memory_bytes": {
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": {
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Set:         schema.HashString,
															},
															"discrete_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The Integer resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Set:         schema.HashString,
															},
														},
													},
												},
											},
										},
									},
									"reservation": {
										Type:        schema.TypeList,
										Description: "An object describing the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": {
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Type:        schema.TypeInt,
													Optional:    true,
												},
												"memory_bytes": {
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": {
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
															"discrete_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The Integer resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"restart_policy": {
							Type:        schema.TypeMap,
							Description: "Specification for the restart policy which applies to containers created as part of this service.",
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"condition": {
										Type:         schema.TypeString,
										Description:  "Condition for restart",
										Optional:     true,
										ValidateFunc: validateStringMatchesPattern(`^(none|on-failure|any)$`),
									},
									"delay": {
										Type:         schema.TypeString,
										Description:  "Delay between restart attempts (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
									"max_attempts": {
										Type:         schema.TypeInt,
										Description:  "Maximum attempts to restart a given container before giving up (default value is 0, which is ignored)",
										Optional:     true,
										ValidateFunc: validateIntegerGeqThan(0),
									},
									"window": {
										Type:         schema.TypeString,
										Description:  "The time window used to evaluate the restart policy (default value is 0, which is unbounded) (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
								},
							},
						},
						"placement": {
							Type:        schema.TypeList,
							Description: "The placement preferences",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"constraints": {
										Type:        schema.TypeSet,
										Description: "An array of constraints. e.g.: node.role==manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"prefs": {
										Type:        schema.TypeSet,
										Description: "Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence, e.g.: spread=node.role.manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"platforms": {
										Type:        schema.TypeSet,
										Description: "Platforms stores all the platforms that the service's image can run on",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"architecture": {
													Type:        schema.TypeString,
													Description: "The architecture, e.g. amd64",
													Required:    true,
												},
												"os": {
													Type:        schema.TypeString,
													Description: "The operation system, e.g. linux",
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
						"force_update": {
							Type:         schema.TypeInt,
							Description:  "A counter that triggers an update even if no relevant parameters have been changed. See https://github.com/docker/swarmkit/blob/master/api/specs.proto#L126",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"runtime": {
							Type:         schema.TypeString,
							Description:  "Runtime is the type of runtime specified for the task executor. See https://github.com/moby/moby/blob/master/api/types/swarm/runtime.go",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern("^(container|plugin)$"),
						},
						"networks": {
							Type:        schema.TypeSet,
							Description: "Ids of the networks in which the  container will be put in.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"log_driver": {
							Type:        schema.TypeList,
							Description: "Specifies the log driver to use for tasks created from this spec. If not present, the default one for the swarm will be used, finally falling back to the engine default if not specified",
							MaxItems:    1,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "The logging driver to use",
										Required:    true,
									},
									"options": {
										Type:        schema.TypeMap,
										Description: "The options for the logging driver",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
			"mode": {
				Type:        schema.TypeList,
				Description: "Scheduling mode for the service",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicated": {
							Type:          schema.TypeList,
							Description:   "The replicated service mode",
							MaxItems:      1,
							Optional:      true,
							Computed:      true,
							ConflictsWith: []string{"mode.0.global"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"replicas": {
										Type:         schema.TypeInt,
										Description:  "The amount of replicas of the service",
										Optional:     true,
										Default:      1,
										ValidateFunc: validateIntegerGeqThan(1),
									},
								},
							},
						},
						"global": {
							Type:          schema.TypeBool,
							Description:   "The global service mode",
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"mode.0.replicated", "converge_config"},
						},
					},
				},
			},
			"update_config": {
				Type:        schema.TypeList,
				Description: "Specification for the update strategy of the service",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": {
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be updated in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": {
							Type:         schema.TypeString,
							Description:  "Delay between task updates (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": {
							Type:         schema.TypeString,
							Description:  "Action on update failure: pause | continue | rollback",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("^(pause|continue|rollback)$"),
						},
						"monitor": {
							Type:         schema.TypeString,
							Description:  "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": {
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during an update",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": {
							Type:         schema.TypeString,
							Description:  "Update order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("^(stop-first|start-first)$"),
						},
					},
				},
			},
			"rollback_config": {
				Type:        schema.TypeList,
				Description: "Specification for the rollback strategy of the service",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": {
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be rollbacked in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": {
							Type:         schema.TypeString,
							Description:  "Delay between task rollbacks (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": {
							Type:         schema.TypeString,
							Description:  "Action on rollback failure: pause | continue",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("(pause|continue)"),
						},
						"monitor": {
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": {
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": {
							Type:         schema.TypeString,
							Description:  "Rollback order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("(stop-first|start-first)"),
						},
					},
				},
			},
			"endpoint_spec": {
				Type:        schema.TypeList,
				Description: "Properties that can be configured to access and load balance a service",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:         schema.TypeString,
							Description:  "The mode of resolution to use for internal load balancing between tasks",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
						},
						"ports": {
							Type:        schema.TypeSet,
							Description: "List of exposed ports that this service is accessible on from the outside. Ports can only be provided if 'vip' resolution mode is used.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "A random name for the port",
										Optional:    true,
									},
									"protocol": {
										Type:         schema.TypeString,
										Description:  "Rrepresents the protocol of a port: 'tcp', 'udp' or 'sctp'",
										Optional:     true,
										Default:      "tcp",
										ValidateFunc: validateStringMatchesPattern(`^(tcp|udp|sctp)$`),
									},
									"target_port": {
										Type:        schema.TypeInt,
										Description: "The port inside the container",
										Required:    true,
									},
									"published_port": {
										Type:        schema.TypeInt,
										Description: "The port on the swarm hosts.",
										Optional:    true,
									},
									"publish_mode": {
										Type:         schema.TypeString,
										Description:  "Represents the mode in which the port is to be published: 'ingress' or 'host'",
										Optional:     true,
										Default:      "ingress",
										ValidateFunc: validateStringMatchesPattern(`^(host|ingress)$`),
									},
								},
							},
						},
					},
				},
			},
			"converge_config": {
				Type:          schema.TypeList,
				Description:   "A configuration to ensure that a service converges aka reaches the desired that of all task up and running",
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{"mode.0.global"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delay": {
							Type:         schema.TypeString,
							Description:  "The interval to check if the desired state is reached (ms|s). Default: 7s",
							Optional:     true,
							Default:      "7s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:         schema.TypeString,
							Description:  "The timeout of the service to reach the desired state (s|m). Default: 3m",
							Optional:     true,
							Default:      "3m",
							ValidateFunc: validateDurationGeq0(),
						},
					},
				},
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerServiceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return migrateServiceLabels(rawState), nil
				},
			},
		},
	}
}

func resourceDockerServiceV0() *schema.Resource {
	return &schema.Resource{
		//This is only used for state migration, so the CRUD
		//callbacks are no longer relevant
		Schema: map[string]*schema.Schema{
			"auth": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server_address": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"username": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
						},
						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Sensitive:   true,
						},
					},
				},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the service",
				Required:    true,
				ForceNew:    true,
			},
			"labels": {
				Type:        schema.TypeMap,
				Description: "User-defined key/value metadata",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"task_spec": {
				Type:        schema.TypeList,
				Description: "User modifiable task configuration",
				MaxItems:    1,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_spec": {
							Type:        schema.TypeList,
							Description: "The spec for each container",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"image": {
										Type:             schema.TypeString,
										Description:      "The image name to use for the containers of the service",
										Required:         true,
										DiffSuppressFunc: suppressIfSHAwasAdded(),
									},
									"labels": {
										Type:        schema.TypeMap,
										Description: "User-defined key/value metadata",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"command": {
										Type:        schema.TypeList,
										Description: "The command to be run in the image",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"args": {
										Type:        schema.TypeList,
										Description: "Arguments to the command",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "The hostname to use for the container, as a valid RFC 1123 hostname",
										Optional:    true,
									},
									"env": {
										Type:        schema.TypeMap,
										Description: "A list of environment variables in the form VAR=\"value\"",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"dir": {
										Type:        schema.TypeString,
										Description: "The working directory for commands to run in",
										Optional:    true,
									},
									"user": {
										Type:        schema.TypeString,
										Description: "The user inside the container",
										Optional:    true,
									},
									"groups": {
										Type:        schema.TypeList,
										Description: "A list of additional groups that the container process will run as",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"privileges": {
										Type:        schema.TypeList,
										Description: "Security options for the container",
										MaxItems:    1,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"credential_spec": {
													Type:        schema.TypeList,
													Description: "CredentialSpec for managed service account (Windows only)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"file": {
																Type:        schema.TypeString,
																Description: "Load credential spec from this file",
																Optional:    true,
															},
															"registry": {
																Type:        schema.TypeString,
																Description: "Load credential spec from this value in the Windows registry",
																Optional:    true,
															},
														},
													},
												},
												"se_linux_context": {
													Type:        schema.TypeList,
													Description: "SELinux labels of the container",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"disable": {
																Type:        schema.TypeBool,
																Description: "Disable SELinux",
																Optional:    true,
															},
															"user": {
																Type:        schema.TypeString,
																Description: "SELinux user label",
																Optional:    true,
															},
															"role": {
																Type:        schema.TypeString,
																Description: "SELinux role label",
																Optional:    true,
															},
															"type": {
																Type:        schema.TypeString,
																Description: "SELinux type label",
																Optional:    true,
															},
															"level": {
																Type:        schema.TypeString,
																Description: "SELinux level label",
																Optional:    true,
															},
														},
													},
												},
											},
										},
									},
									"read_only": {
										Type:        schema.TypeBool,
										Description: "Mount the container's root filesystem as read only",
										Optional:    true,
									},
									"mounts": {
										Type:        schema.TypeSet,
										Description: "Specification for mounts to be added to containers created as part of the service",
										Optional:    true,
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
													Type:         schema.TypeString,
													Description:  "The mount type",
													Required:     true,
													ValidateFunc: validateStringMatchesPattern(`^(bind|volume|tmpfs)$`),
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
																Type:         schema.TypeString,
																Description:  "A propagation mode with the value",
																Optional:     true,
																ValidateFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
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
																Type:        schema.TypeMap,
																Description: "User-defined key/value metadata",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
															"driver_name": {
																Type:        schema.TypeString,
																Description: "Name of the driver to use to create the volume.",
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
									"stop_signal": {
										Type:        schema.TypeString,
										Description: "Signal to stop the container",
										Optional:    true,
									},
									"stop_grace_period": {
										Type:         schema.TypeString,
										Description:  "Amount of time to wait for the container to terminate before forcefully removing it (ms|s|m|h)",
										Optional:     true,
										Computed:     true,
										ValidateFunc: validateDurationGeq0(),
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
									"hosts": {
										Type:        schema.TypeSet,
										Description: "A list of hostname/IP mappings to add to the container's hosts file.",
										Optional:    true,
										ForceNew:    true,
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
									"dns_config": {
										Type:        schema.TypeList,
										Description: "Specification for DNS related configurations in resolver configuration file (resolv.conf)",
										MaxItems:    1,
										Optional:    true,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nameservers": {
													Type:        schema.TypeList,
													Description: "The IP addresses of the name servers",
													Required:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"search": {
													Type:        schema.TypeList,
													Description: "A search list for host-name lookup",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"options": {
													Type:        schema.TypeList,
													Description: "A list of internal resolver variables to be modified (e.g., debug, ndots:3, etc.)",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"secrets": {
										Type:        schema.TypeSet,
										Description: "References to zero or more secrets that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"secret_id": {
													Type:        schema.TypeString,
													Description: "ID of the specific secret that we're referencing",
													Required:    true,
												},
												"secret_name": {
													Type:        schema.TypeString,
													Description: "Name of the secret that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": {
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"configs": {
										Type:        schema.TypeSet,
										Description: "References to zero or more configs that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"config_id": {
													Type:        schema.TypeString,
													Description: "ID of the specific config that we're referencing",
													Required:    true,
												},
												"config_name": {
													Type:        schema.TypeString,
													Description: "Name of the config that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": {
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"isolation": {
										Type:         schema.TypeString,
										Description:  "Isolation technology of the containers running the service. (Windows only)",
										Optional:     true,
										Default:      "default",
										ValidateFunc: validateStringMatchesPattern(`^(default|process|hyperv)$`),
									},
								},
							},
						},
						"resources": {
							Type:        schema.TypeList,
							Description: "Resource requirements which apply to each individual container created as part of the service",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"limits": {
										Type:        schema.TypeList,
										Description: "Describes the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": {
													Type:        schema.TypeInt,
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Optional:    true,
												},
												"memory_bytes": {
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": {
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Set:         schema.HashString,
															},
															"discrete_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The Integer resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Set:         schema.HashString,
															},
														},
													},
												},
											},
										},
									},
									"reservation": {
										Type:        schema.TypeList,
										Description: "An object describing the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": {
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Type:        schema.TypeInt,
													Optional:    true,
												},
												"memory_bytes": {
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": {
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
															"discrete_resources_spec": {
																Type:        schema.TypeSet,
																Description: "The Integer resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"restart_policy": {
							Type:        schema.TypeMap,
							Description: "Specification for the restart policy which applies to containers created as part of this service.",
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"condition": {
										Type:         schema.TypeString,
										Description:  "Condition for restart",
										Optional:     true,
										ValidateFunc: validateStringMatchesPattern(`^(none|on-failure|any)$`),
									},
									"delay": {
										Type:         schema.TypeString,
										Description:  "Delay between restart attempts (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
									"max_attempts": {
										Type:         schema.TypeInt,
										Description:  "Maximum attempts to restart a given container before giving up (default value is 0, which is ignored)",
										Optional:     true,
										ValidateFunc: validateIntegerGeqThan(0),
									},
									"window": {
										Type:         schema.TypeString,
										Description:  "The time window used to evaluate the restart policy (default value is 0, which is unbounded) (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
								},
							},
						},
						"placement": {
							Type:        schema.TypeList,
							Description: "The placement preferences",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"constraints": {
										Type:        schema.TypeSet,
										Description: "An array of constraints. e.g.: node.role==manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"prefs": {
										Type:        schema.TypeSet,
										Description: "Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence, e.g.: spread=node.role.manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"platforms": {
										Type:        schema.TypeSet,
										Description: "Platforms stores all the platforms that the service's image can run on",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"architecture": {
													Type:        schema.TypeString,
													Description: "The architecture, e.g. amd64",
													Required:    true,
												},
												"os": {
													Type:        schema.TypeString,
													Description: "The operation system, e.g. linux",
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
						"force_update": {
							Type:         schema.TypeInt,
							Description:  "A counter that triggers an update even if no relevant parameters have been changed. See https://github.com/docker/swarmkit/blob/master/api/specs.proto#L126",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"runtime": {
							Type:         schema.TypeString,
							Description:  "Runtime is the type of runtime specified for the task executor. See https://github.com/moby/moby/blob/master/api/types/swarm/runtime.go",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern("^(container|plugin)$"),
						},
						"networks": {
							Type:        schema.TypeSet,
							Description: "Ids of the networks in which the  container will be put in.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"log_driver": {
							Type:        schema.TypeList,
							Description: "Specifies the log driver to use for tasks created from this spec. If not present, the default one for the swarm will be used, finally falling back to the engine default if not specified",
							MaxItems:    1,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "The logging driver to use",
										Required:    true,
									},
									"options": {
										Type:        schema.TypeMap,
										Description: "The options for the logging driver",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
			"mode": {
				Type:        schema.TypeList,
				Description: "Scheduling mode for the service",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicated": {
							Type:          schema.TypeList,
							Description:   "The replicated service mode",
							MaxItems:      1,
							Optional:      true,
							Computed:      true,
							ConflictsWith: []string{"mode.0.global"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"replicas": {
										Type:         schema.TypeInt,
										Description:  "The amount of replicas of the service",
										Optional:     true,
										Default:      1,
										ValidateFunc: validateIntegerGeqThan(1),
									},
								},
							},
						},
						"global": {
							Type:          schema.TypeBool,
							Description:   "The global service mode",
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"mode.0.replicated", "converge_config"},
						},
					},
				},
			},
			"update_config": {
				Type:        schema.TypeList,
				Description: "Specification for the update strategy of the service",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": {
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be updated in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": {
							Type:         schema.TypeString,
							Description:  "Delay between task updates (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": {
							Type:         schema.TypeString,
							Description:  "Action on update failure: pause | continue | rollback",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("^(pause|continue|rollback)$"),
						},
						"monitor": {
							Type:         schema.TypeString,
							Description:  "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": {
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during an update",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": {
							Type:         schema.TypeString,
							Description:  "Update order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("^(stop-first|start-first)$"),
						},
					},
				},
			},
			"rollback_config": {
				Type:        schema.TypeList,
				Description: "Specification for the rollback strategy of the service",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": {
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be rollbacked in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": {
							Type:         schema.TypeString,
							Description:  "Delay between task rollbacks (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": {
							Type:         schema.TypeString,
							Description:  "Action on rollback failure: pause | continue",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("(pause|continue)"),
						},
						"monitor": {
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": {
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": {
							Type:         schema.TypeString,
							Description:  "Rollback order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("(stop-first|start-first)"),
						},
					},
				},
			},
			"endpoint_spec": {
				Type:        schema.TypeList,
				Description: "Properties that can be configured to access and load balance a service",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:         schema.TypeString,
							Description:  "The mode of resolution to use for internal load balancing between tasks",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
						},
						"ports": {
							Type:        schema.TypeSet,
							Description: "List of exposed ports that this service is accessible on from the outside. Ports can only be provided if 'vip' resolution mode is used.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "A random name for the port",
										Optional:    true,
									},
									"protocol": {
										Type:         schema.TypeString,
										Description:  "Rrepresents the protocol of a port: 'tcp', 'udp' or 'sctp'",
										Optional:     true,
										Default:      "tcp",
										ValidateFunc: validateStringMatchesPattern(`^(tcp|udp|sctp)$`),
									},
									"target_port": {
										Type:        schema.TypeInt,
										Description: "The port inside the container",
										Required:    true,
									},
									"published_port": {
										Type:        schema.TypeInt,
										Description: "The port on the swarm hosts.",
										Optional:    true,
									},
									"publish_mode": {
										Type:         schema.TypeString,
										Description:  "Represents the mode in which the port is to be published: 'ingress' or 'host'",
										Optional:     true,
										Default:      "ingress",
										ValidateFunc: validateStringMatchesPattern(`^(host|ingress)$`),
									},
								},
							},
						},
					},
				},
			},
			"converge_config": {
				Type:          schema.TypeList,
				Description:   "A configuration to ensure that a service converges aka reaches the desired that of all task up and running",
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{"mode.0.global"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delay": {
							Type:         schema.TypeString,
							Description:  "The interval to check if the desired state is reached (ms|s). Default: 7s",
							Optional:     true,
							Default:      "7s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:         schema.TypeString,
							Description:  "The timeout of the service to reach the desired state (s|m). Default: 3m",
							Optional:     true,
							Default:      "3m",
							ValidateFunc: validateDurationGeq0(),
						},
					},
				},
			},
		},
	}
}

func suppressIfSHAwasAdded() schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		// the initial case when the service is created
		if old == "" && new != "" {
			return false
		}

		oldURL, oldImage, oldTag, oldDigest, oldErr := splitImageName(old)
		if oldErr != nil {
			log.Printf("[DEBUG] invalid old image name: %s\n", oldErr.Error())
			return false
		}
		log.Printf("[DEBUG] old image parse: %s, %s, %s, %s\n", oldURL, oldImage, oldTag, oldDigest)

		newURL, newImage, newTag, newDigest, newErr := splitImageName(new)
		if newErr != nil {
			log.Printf("[DEBUG] invalid new image name: %s\n", newErr.Error())
			return false
		}
		log.Printf("[DEBUG] new image parse: %s, %s, %s, %s\n", newURL, newImage, newTag, newDigest)

		if oldURL != newURL || oldImage != newImage {
			return false
		}

		// special case with latest
		if oldTag == "latest" && (newTag == "" || newTag == "latest") {
			if oldDigest != "" && newDigest == "" {
				return true
			}

			return false
		}

		// https://success.docker.com/article/images-tagging-vs-digests
		// we always pull if the tag changes, also in the empty and 'latest' case
		if (oldTag == "latest" || newTag == "") || (oldTag == "" && newTag == "latest") {
			return false
		}

		if oldTag != newTag {
			return false
		}

		// tags are the same and so should be its digests
		if oldDigest == newDigest || (oldDigest == "" && newDigest != "") || (oldDigest != "" && newDigest == "") {
			return true
		}

		// we only update if the digests are given and different
		if oldDigest != newDigest {
			return false
		}

		return true
	}
}

// spitImageName splits an image with name 127.0.0.1:15000/tftest-service:v1@sha256:24..
// into its parts. Handles edge cases like no tag and no digest
func splitImageName(imageNameToSplit string) (url, image, tag, digest string, err error) {
	urlToRestSplit := strings.Split(imageNameToSplit, "/")
	if len(urlToRestSplit) != 2 {
		return "", "", "", "", fmt.Errorf("image name is not valid: %s", imageNameToSplit)
	}
	url = urlToRestSplit[0]
	imageNameToRestSplit := strings.Split(urlToRestSplit[1], ":")
	// we only have an image name without tag and sha256
	if len(imageNameToRestSplit) == 1 {
		image = imageNameToRestSplit[0]
		return url, image, "", "", nil
	}
	// has tag and sha256
	if len(imageNameToRestSplit) == 3 {
		image = imageNameToRestSplit[0]
		tag = strings.Replace(imageNameToRestSplit[1], "@sha256", "", 1)
		digest = imageNameToRestSplit[2]
		return url, image, tag, digest, nil
	}
	// can be either with tag or sha256, which implies 'latest' tag
	if len(imageNameToRestSplit) == 2 {
		image = imageNameToRestSplit[0]
		if strings.Contains(imageNameToRestSplit[1], "sha256") {
			digest = imageNameToRestSplit[1]
			return url, image, "", digest, nil
		}
		tag = strings.Replace(imageNameToRestSplit[1], "@sha256", "", 1)
		return url, image, tag, "", nil
	}

	return "", "", "", "", fmt.Errorf("image name is not valid: %s", imageNameToSplit)
}
