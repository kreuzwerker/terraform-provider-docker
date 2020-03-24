package docker

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerRegistryImageCreate,
		Read:   resourceDockerRegistryImageRead,
		Delete: resourceDockerRegistryImageDelete,
		Update: resourceDockerRegistryImageUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"keep_remotely": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"build": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"suppress_output": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"remote_context": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"no_cache": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"remove": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"force_remove": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"pull_parent": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"isolation": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_set_cpus": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_set_mems": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_shares": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cpu_quota": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cpu_period": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"memory": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"memory_swap": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cgroup_parent": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"network_mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"shm_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"dockerfile": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Dockerfile",
							ForceNew: true,
						},
						"ulimit": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"hard": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},
									"soft": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},
								},
							},
						},
						"build_args": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"auth_config": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"user_name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"password": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"auth": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"email": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"server_address": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"identity_token": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"registry_token": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"context": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
							StateFunc: func(val interface{}) string {
								// the context hash is stored to identify changes in the context files
								dockerContextTarPath, _ := buildDockerImageContextTar(val.(string))
								defer os.Remove(dockerContextTarPath)
								contextTarHash, _ := getDockerImageContextTarHash(dockerContextTarPath)
								return val.(string) + ":" + contextTarHash
							},
						},
						"labels": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"squash": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"cache_from": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"security_opt": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"extra_hosts": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"target": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"session_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"platform": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"version": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"build_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						// "output": &schema.Schema{
						// 	Type:     schema.TypeString,
						// 	Optional: true,
						// },
					},
				},
			},

			"sha256_digest": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
