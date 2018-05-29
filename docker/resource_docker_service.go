package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
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

		Schema: map[string]*schema.Schema{
			"auth": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"username": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
						},
						"password": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Sensitive:   true,
						},
					},
				},
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Name of the service",
				Required:    true,
				ForceNew:    true,
			},
			"labels": &schema.Schema{
				Type:        schema.TypeMap,
				Description: "User-defined key/value metadata",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"task_spec": &schema.Schema{
				Type:        schema.TypeList,
				Description: "User modifiable task configuration",
				MaxItems:    1,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_spec": &schema.Schema{
							Type:        schema.TypeList,
							Description: "The spec for each container",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"image": &schema.Schema{
										Type:        schema.TypeString,
										Description: "The image name to use for the containers of the service",
										Required:    true,
									},
									"labels": &schema.Schema{
										Type:        schema.TypeMap,
										Description: "User-defined key/value metadata",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"command": &schema.Schema{
										Type:        schema.TypeList,
										Description: "The command to be run in the image",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"args": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Arguments to the command",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"hostname": &schema.Schema{
										Type:        schema.TypeString,
										Description: "The hostname to use for the container, as a valid RFC 1123 hostname",
										Optional:    true,
									},
									"env": &schema.Schema{
										Type:        schema.TypeMap,
										Description: "A list of environment variables in the form VAR=\"value\"",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"dir": &schema.Schema{
										Type:        schema.TypeString,
										Description: "The working directory for commands to run in",
										Optional:    true,
									},
									"user": &schema.Schema{
										Type:        schema.TypeString,
										Description: "The user inside the container",
										Optional:    true,
									},
									"groups": &schema.Schema{
										Type:        schema.TypeList,
										Description: "A list of additional groups that the container process will run as",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"privileges": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Security options for the container",
										MaxItems:    1,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"credential_spec": &schema.Schema{
													Type:        schema.TypeList,
													Description: "CredentialSpec for managed service account (Windows only)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"file": &schema.Schema{
																Type:        schema.TypeString,
																Description: "Load credential spec from this file",
																Optional:    true,
															},
															"registry": &schema.Schema{
																Type:        schema.TypeString,
																Description: "Load credential spec from this value in the Windows registry",
																Optional:    true,
															},
														},
													},
												},
												"se_linux_context": &schema.Schema{
													Type:        schema.TypeList,
													Description: "SELinux labels of the container",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"disable": &schema.Schema{
																Type:        schema.TypeBool,
																Description: "Disable SELinux",
																Optional:    true,
															},
															"user": &schema.Schema{
																Type:        schema.TypeString,
																Description: "SELinux user label",
																Optional:    true,
															},
															"role": &schema.Schema{
																Type:        schema.TypeString,
																Description: "SELinux role label",
																Optional:    true,
															},
															"type": &schema.Schema{
																Type:        schema.TypeString,
																Description: "SELinux type label",
																Optional:    true,
															},
															"level": &schema.Schema{
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
									"read_only": &schema.Schema{
										Type:        schema.TypeBool,
										Description: "Mount the container's root filesystem as read only",
										Optional:    true,
									},
									"mounts": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "Specification for mounts to be added to containers created as part of the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"target": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Container path",
													Required:    true,
												},
												"source": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Mount source (e.g. a volume name, a host path)",
													Required:    true,
												},
												"type": &schema.Schema{
													Type:         schema.TypeString,
													Description:  "The mount type",
													Required:     true,
													ValidateFunc: validateStringMatchesPattern(`^(bind|volume|tmpfs)$`),
												},
												"read_only": &schema.Schema{
													Type:        schema.TypeBool,
													Description: "Whether the mount should be read-only",
													Optional:    true,
												},
												"bind_options": &schema.Schema{
													Type:        schema.TypeList,
													Description: "Optional configuration for the bind type",
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"propagation": &schema.Schema{
																Type:         schema.TypeString,
																Description:  "A propagation mode with the value",
																Optional:     true,
																ValidateFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
															},
														},
													},
												},
												"volume_options": &schema.Schema{
													Type:        schema.TypeList,
													Description: "Optional configuration for the volume type",
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"no_copy": &schema.Schema{
																Type:        schema.TypeBool,
																Description: "Populate volume with data from the target",
																Optional:    true,
															},
															"labels": &schema.Schema{
																Type:        schema.TypeMap,
																Description: "User-defined key/value metadata",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
															"driver_name": &schema.Schema{
																Type:        schema.TypeString,
																Description: "Name of the driver to use to create the volume.",
																Optional:    true,
															},
															"driver_options": &schema.Schema{
																Type:        schema.TypeMap,
																Description: "key/value map of driver specific options",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
												"tmpfs_options": &schema.Schema{
													Type:        schema.TypeList,
													Description: "Optional configuration for the tmpfs type",
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"size_bytes": &schema.Schema{
																Type:        schema.TypeInt,
																Description: "The size for the tmpfs mount in bytes",
																Optional:    true,
															},
															"mode": &schema.Schema{
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
									"stop_signal": &schema.Schema{
										Type:        schema.TypeString,
										Description: "Signal to stop the container",
										Optional:    true,
									},
									"stop_grace_period": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "Amount of time to wait for the container to terminate before forcefully removing it (ms|s|m|h)",
										Optional:     true,
										Computed:     true,
										ValidateFunc: validateDurationGeq0(),
									},
									"healthcheck": &schema.Schema{
										Type:        schema.TypeList,
										Description: "A test to perform to check that the container is healthy",
										MaxItems:    1,
										Optional:    true,
										Computed:    true,
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
									"hosts": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "A list of hostname/IP mappings to add to the container's hosts file.",
										Optional:    true,
										ForceNew:    true,
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
									"dns_config": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Specification for DNS related configurations in resolver configuration file (resolv.conf)",
										MaxItems:    1,
										Optional:    true,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nameservers": &schema.Schema{
													Type:        schema.TypeList,
													Description: "The IP addresses of the name servers",
													Required:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"search": &schema.Schema{
													Type:        schema.TypeList,
													Description: "A search list for host-name lookup",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"options": &schema.Schema{
													Type:        schema.TypeList,
													Description: "A list of internal resolver variables to be modified (e.g., debug, ndots:3, etc.)",
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"secrets": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "References to zero or more secrets that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"secret_id": &schema.Schema{
													Type:        schema.TypeString,
													Description: "ID of the specific secret that we're referencing",
													Required:    true,
												},
												"secret_name": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Name of the secret that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"configs": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "References to zero or more configs that will be exposed to the service",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"config_id": &schema.Schema{
													Type:        schema.TypeString,
													Description: "ID of the specific config that we're referencing",
													Required:    true,
												},
												"config_name": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Name of the config that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID",
													Optional:    true,
												},
												"file_name": &schema.Schema{
													Type:        schema.TypeString,
													Description: "Represents the final filename in the filesystem",
													Required:    true,
												},
											},
										},
									},
									"isolation": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "Isolation technology of the containers running the service. (Windows only)",
										Optional:     true,
										Default:      "default",
										ValidateFunc: validateStringMatchesPattern(`^(default|process|hyperv)$`),
									},
								},
							},
						},
						"resources": &schema.Schema{
							Type:        schema.TypeList,
							Description: "Resource requirements which apply to each individual container created as part of the service",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"limits": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Describes the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": &schema.Schema{
													Type:        schema.TypeInt,
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Optional:    true,
												},
												"memory_bytes": &schema.Schema{
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": &schema.Schema{
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": &schema.Schema{
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Set:         schema.HashString,
															},
															"discrete_resources_spec": &schema.Schema{
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
									"reservation": &schema.Schema{
										Type:        schema.TypeList,
										Description: "An object describing the resources which can be advertised by a node and requested by a task",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nano_cpus": &schema.Schema{
													Description: "CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000",
													Type:        schema.TypeInt,
													Optional:    true,
												},
												"memory_bytes": &schema.Schema{
													Type:        schema.TypeInt,
													Description: "The amounf of memory in bytes the container allocates",
													Optional:    true,
												},
												"generic_resources": &schema.Schema{
													Type:        schema.TypeList,
													Description: "User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)",
													MaxItems:    1,
													Optional:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"named_resources_spec": &schema.Schema{
																Type:        schema.TypeSet,
																Description: "The String resources",
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
															"discrete_resources_spec": &schema.Schema{
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
						"restart_policy": &schema.Schema{
							Type:        schema.TypeMap,
							Description: "Specification for the restart policy which applies to containers created as part of this service.",
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"condition": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "Condition for restart",
										Optional:     true,
										ValidateFunc: validateStringMatchesPattern(`^(none|on-failure|any)$`),
									},
									"delay": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "Delay between restart attempts (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
									"max_attempts": &schema.Schema{
										Type:         schema.TypeInt,
										Description:  "Maximum attempts to restart a given container before giving up (default value is 0, which is ignored)",
										Optional:     true,
										ValidateFunc: validateIntegerGeqThan(0),
									},
									"window": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "The time window used to evaluate the restart policy (default value is 0, which is unbounded) (ms|s|m|h)",
										Optional:     true,
										ValidateFunc: validateDurationGeq0(),
									},
								},
							},
						},
						"placement": &schema.Schema{
							Type:        schema.TypeList,
							Description: "The placement preferences",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"constraints": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "An array of constraints. e.g.: node.role==manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"prefs": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence, e.g.: spread=node.role.manager",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Set:         schema.HashString,
									},
									"platforms": &schema.Schema{
										Type:        schema.TypeSet,
										Description: "Platforms stores all the platforms that the service's image can run on",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"architecture": &schema.Schema{
													Type:        schema.TypeString,
													Description: "The architecture, e.g. amd64",
													Required:    true,
												},
												"os": &schema.Schema{
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
						"force_update": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "A counter that triggers an update even if no relevant parameters have been changed. See https://github.com/docker/swarmkit/blob/master/api/specs.proto#L126",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"runtime": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Runtime is the type of runtime specified for the task executor. See https://github.com/moby/moby/blob/master/api/types/swarm/runtime.go",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern("^(container|plugin)$"),
						},
						"networks": &schema.Schema{
							Type:        schema.TypeSet,
							Description: "Ids of the networks in which the  container will be put in.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"log_driver": &schema.Schema{
							Type:        schema.TypeList,
							Description: "Specifies the log driver to use for tasks created from this spec. If not present, the default one for the swarm will be used, finally falling back to the engine default if not specified",
							MaxItems:    1,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "The logging driver to use: one of none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs",
										Required:     true,
										ValidateFunc: validateStringMatchesPattern("(none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs)"),
									},
									"options": &schema.Schema{
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
			"mode": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Scheduling mode for the service",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicated": &schema.Schema{
							Type:          schema.TypeList,
							Description:   "The replicated service mode",
							MaxItems:      1,
							Optional:      true,
							Computed:      true,
							ConflictsWith: []string{"mode.0.global"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"replicas": &schema.Schema{
										Type:         schema.TypeInt,
										Description:  "The amount of replicas of the service",
										Optional:     true,
										Default:      1,
										ValidateFunc: validateIntegerGeqThan(1),
									},
								},
							},
						},
						"global": &schema.Schema{
							Type:          schema.TypeBool,
							Description:   "The global service mode",
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"mode.0.replicated", "converge_config"},
						},
					},
				},
			},
			"update_config": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Specification for the update strategy of the service",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be updated in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between task updates (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on update failure: pause | continue | rollback",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("^(pause|continue|rollback)$"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during an update",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Update order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("^(stop-first|start-first)$"),
						},
					},
				},
			},
			"rollback_config": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Specification for the rollback strategy of the service",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be rollbacked in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between task rollbacks (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on rollback failure: pause | continue",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("(pause|continue)"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Rollback order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("(stop-first|start-first)"),
						},
					},
				},
			},
			"endpoint_spec": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Properties that can be configured to access and load balance a service",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "The mode of resolution to use for internal load balancing between tasks",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
						},
						"ports": &schema.Schema{
							Type:        schema.TypeSet,
							Description: "List of exposed ports that this service is accessible on from the outside. Ports can only be provided if 'vip' resolution mode is used.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:        schema.TypeString,
										Description: "A random name for the port",
										Optional:    true,
									},
									"protocol": &schema.Schema{
										Type:         schema.TypeString,
										Description:  "Rrepresents the protocol of a port: 'tcp', 'udp' or 'sctp'",
										Optional:     true,
										Default:      "tcp",
										ValidateFunc: validateStringMatchesPattern(`^(tcp|udp|sctp)$`),
									},
									"target_port": &schema.Schema{
										Type:        schema.TypeInt,
										Description: "The port inside the container",
										Required:    true,
									},
									"published_port": &schema.Schema{
										Type:        schema.TypeInt,
										Description: "The port on the swarm hosts.",
										Optional:    true,
									},
									"publish_mode": &schema.Schema{
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
			"converge_config": &schema.Schema{
				Type:          schema.TypeList,
				Description:   "A configuration to ensure that a service converges aka reaches the desired that of all task up and running",
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{"mode.0.global"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "The interval to check if the desired state is reached (ms|s). Default: 7s",
							Optional:     true,
							Default:      "7s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": &schema.Schema{
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
