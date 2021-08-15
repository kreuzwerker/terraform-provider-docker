package provider

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the lifecycle of docker image/tag in a registry.",

		CreateContext: resourceDockerRegistryImageCreate,
		ReadContext:   resourceDockerRegistryImageRead,
		DeleteContext: resourceDockerRegistryImageDelete,
		UpdateContext: resourceDockerRegistryImageUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image.",
				Required:    true,
				ForceNew:    true,
			},

			"keep_remotely": {
				Type:        schema.TypeBool,
				Description: "If true, then the Docker image won't be deleted on destroy operation. If this is false, it will delete the image from the docker registry on destroy operation. Defaults to `false`",
				Default:     false,
				Optional:    true,
			},

			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Description: "If `true`, the verification of TLS certificates of the server/registry is disabled. Defaults to `false`",
				Optional:    true,
				Default:     false,
			},

			"build": {
				Type:        schema.TypeList,
				Description: "Definition for building the image",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"suppress_output": {
							Type:        schema.TypeBool,
							Description: "Suppress the build output and print image ID on success",
							Optional:    true,
							ForceNew:    true,
						},
						"remote_context": {
							Type:        schema.TypeString,
							Description: "A Git repository URI or HTTP/HTTPS context URI",
							Optional:    true,
							ForceNew:    true,
						},
						"no_cache": {
							Type:        schema.TypeBool,
							Description: "Do not use the cache when building the image",
							Optional:    true,
							ForceNew:    true,
						},
						"remove": {
							Type:        schema.TypeBool,
							Description: "Remove intermediate containers after a successful build (default behavior)",
							Optional:    true,
							ForceNew:    true,
						},
						"force_remove": {
							Type:        schema.TypeBool,
							Description: "Always remove intermediate containers",
							Optional:    true,
							ForceNew:    true,
						},
						"pull_parent": {
							Type:        schema.TypeBool,
							Description: "Attempt to pull the image even if an older image exists locally",
							Optional:    true,
							ForceNew:    true,
						},
						"isolation": {
							Type:        schema.TypeString,
							Description: "Isolation represents the isolation technology of a container. The supported values are ",
							Optional:    true,
							ForceNew:    true,
						},
						"cpu_set_cpus": {
							Type:        schema.TypeString,
							Description: "CPUs in which to allow execution (e.g., `0-3`, `0`, `1`)",
							Optional:    true,
							ForceNew:    true,
						},
						"cpu_set_mems": {
							Type:        schema.TypeString,
							Description: "MEMs in which to allow execution (`0-3`, `0`, `1`)",
							Optional:    true,
							ForceNew:    true,
						},
						"cpu_shares": {
							Type:        schema.TypeInt,
							Description: "CPU shares (relative weight)",
							Optional:    true,
							ForceNew:    true,
						},
						"cpu_quota": {
							Type:        schema.TypeInt,
							Description: "Microseconds of CPU time that the container can get in a CPU period",
							Optional:    true,
							ForceNew:    true,
						},
						"cpu_period": {
							Type:        schema.TypeInt,
							Description: "The length of a CPU period in microseconds",
							Optional:    true,
							ForceNew:    true,
						},
						"memory": {
							Type:        schema.TypeInt,
							Description: "Set memory limit for build",
							Optional:    true,
							ForceNew:    true,
						},
						"memory_swap": {
							Type:        schema.TypeInt,
							Description: "Total memory (memory + swap), -1 to enable unlimited swap",
							Optional:    true,
							ForceNew:    true,
						},
						"cgroup_parent": {
							Type:        schema.TypeString,
							Description: "Optional parent cgroup for the container",
							Optional:    true,
							ForceNew:    true,
						},
						"network_mode": {
							Type:        schema.TypeString,
							Description: "Set the networking mode for the RUN instructions during build",
							Optional:    true,
							ForceNew:    true,
						},
						"shm_size": {
							Type:        schema.TypeInt,
							Description: "Size of /dev/shm in bytes. The size must be greater than 0",
							Optional:    true,
							ForceNew:    true,
						},
						"dockerfile": {
							Type:        schema.TypeString,
							Description: "Dockerfile file. Defaults to `Dockerfile`",
							Default:     "Dockerfile",
							Optional:    true,
							ForceNew:    true,
						},
						"ulimit": {
							Type:        schema.TypeList,
							Description: "Configuration for ulimits",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "type of ulimit, e.g. `nofile`",
										Required:    true,
										ForceNew:    true,
									},
									"hard": {
										Type:        schema.TypeInt,
										Description: "soft limit",
										Required:    true,
										ForceNew:    true,
									},
									"soft": {
										Type:        schema.TypeInt,
										Description: "hard limit",
										Required:    true,
										ForceNew:    true,
									},
								},
							},
						},
						"build_args": {
							Type:        schema.TypeMap,
							Description: "Pairs for build-time variables in the form TODO",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "The argument",
							},
						},
						"auth_config": {
							Type:        schema.TypeList,
							Description: "The configuration for the authentication",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_name": {
										Type:        schema.TypeString,
										Description: "hostname of the registry",
										Required:    true,
									},
									"user_name": {
										Type:        schema.TypeString,
										Description: "the registry user name",
										Optional:    true,
									},
									"password": {
										Type:        schema.TypeString,
										Description: "the registry password",
										Optional:    true,
									},
									"auth": {
										Type:        schema.TypeString,
										Description: "the auth token",
										Optional:    true,
									},
									"email": {
										Type:        schema.TypeString,
										Description: "the user emal",
										Optional:    true,
									},
									"server_address": {
										Type:        schema.TypeString,
										Description: "the server address",
										Optional:    true,
									},
									"identity_token": {
										Type:        schema.TypeString,
										Description: "the identity token",
										Optional:    true,
									},
									"registry_token": {
										Type:        schema.TypeString,
										Description: "the registry token",
										Optional:    true,
									},
								},
							},
						},
						"context": {
							Type:        schema.TypeString,
							Description: "The absolute path to the context folder. You can use the helper function '${path.cwd}/context-dir'.",
							Required:    true,
							ForceNew:    true,
							StateFunc: func(val interface{}) string {
								// the context hash is stored to identify changes in the context files
								dockerContextTarPath, _ := buildDockerImageContextTar(val.(string))
								defer os.Remove(dockerContextTarPath)
								contextTarHash, _ := getDockerImageContextTarHash(dockerContextTarPath)
								return val.(string) + ":" + contextTarHash
							},
						},
						"labels": {
							Type:        schema.TypeMap,
							Description: "User-defined key/value metadata",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "The key/value pair",
							},
						},
						"squash": {
							Type:        schema.TypeBool,
							Description: "If true the new layers are squashed into a new image with a single new layer",
							Optional:    true,
							ForceNew:    true,
						},
						"cache_from": {
							Type:        schema.TypeList,
							Description: "Images to consider as cache sources",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "The image",
							},
						},
						"security_opt": {
							Type:        schema.TypeList,
							Description: "The security options",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "The option",
							},
						},
						"extra_hosts": {
							Type:        schema.TypeList,
							Description: "A list of hostnames/IP mappings to add to the container’s /etc/hosts file. Specified in the form [\"hostname:IP\"]",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "",
							},
						},
						"target": {
							Type:        schema.TypeString,
							Description: "Set the target build stage to build",
							Optional:    true,
							ForceNew:    true,
						},
						"session_id": {
							Type:        schema.TypeString,
							Description: "Set an ID for the build session",
							Optional:    true,
							ForceNew:    true,
						},
						"platform": {
							Type:        schema.TypeString,
							Description: "Set platform if server is multi-platform capable",
							Optional:    true,
							ForceNew:    true,
						},
						"version": {
							Type:        schema.TypeString,
							Description: "Version of the unerlying builder to use",
							Optional:    true,
							ForceNew:    true,
						},
						"build_id": {
							Type:        schema.TypeString,
							Description: "BuildID is an optional identifier that can be passed together with the build request. The ",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},

			"sha256_digest": {
				Type:        schema.TypeString,
				Description: "The sha256 digest of the image.",
				Computed:    true,
			},
		},
	}
}
