package docker

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

			"build": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"suppress_output": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"remote_context": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"no_cache": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"remove": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"force_remove": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"pull_parent": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"isolation": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_set_cpus": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_set_mems": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"cpu_shares": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cpu_quota": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cpu_period": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"memory": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"memory_swap": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"cgroup_parent": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"network_mode": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"shm_size": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"dockerfile": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Dockerfile",
							ForceNew: true,
						},
						"ulimit": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"hard": {
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},
									"soft": {
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},
								},
							},
						},
						"build_args": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"auth_config": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"user_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"password": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"auth": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"email": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"server_address": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"identity_token": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"registry_token": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"context": {
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
						"labels": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"squash": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"cache_from": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"security_opt": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"extra_hosts": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"target": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"session_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"platform": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"version": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"build_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
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
