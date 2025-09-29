package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	// import drivers otherwise factories are empty
	// for --driver output flag usage
	_ "github.com/docker/buildx/driver/docker"
	_ "github.com/docker/buildx/driver/docker-container"
	_ "github.com/docker/buildx/driver/kubernetes"
	_ "github.com/docker/buildx/driver/remote"
)

// resourceDockerBuildxBuilder defines the buildx_builder resource schema
func resourceDockerBuildxBuilder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDockerBuildxBuilderCreate,
		ReadContext:   resourceDockerBuildxBuilderRead,
		DeleteContext: resourceDockerBuildxBuilderDelete,
		Description:   "Manages a Docker Buildx builder instance. This resource allows you to create a  buildx builder with various configurations such as driver, nodes, and platform settings. Please see https://github.com/docker/buildx/blob/master/docs/reference/buildx_create.md for more documentation",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Default:     "",
				Description: "The name of the Buildx builder. IF not specified, a random name will be generated.",
				ForceNew:    true,
				Optional:    true,
			},
			"driver": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "docker-container",
				Description:   "The driver to use for the Buildx builder (e.g., docker-container, kubernetes).",
				ConflictsWith: []string{"docker_container", "kubernetes", "remote"},
				ForceNew:      true,
			},
			"driver_options": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional options for the Buildx driver in the form of `key=value,...`. These options are driver-specific.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"docker_container", "kubernetes", "remote"},
				ForceNew:      true,
			},
			"node": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Create/modify node with given name",
				Default:     "",
				ForceNew:    true,
			},
			"platform": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Fixed platforms for current node",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ForceNew: true,
			},
			"buildkit_flags": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BuildKit flags to set for the builder.",
				Default:     "",
				ForceNew:    true,
			},
			"buildkit_config": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BuildKit daemon config file",
				Default:     "",
				ForceNew:    true,
			},
			"use": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set the current builder instance as the default for the current context.",
				ForceNew:    true,
			},
			"append": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Append a node to builder instead of changing it",
				ForceNew:    true,
			},
			"bootstrap": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Automatically boot the builder after creation. Defaults to `false`",
				ForceNew:    true,
			},
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The endpoint or context to use for the Buildx builder, where context is the name of a context from docker context ls and endpoint is the address for Docker socket (eg. DOCKER_HOST value). By default, the current Docker configuration is used for determining the context/endpoint value.",
				Default:     "",
				ForceNew:    true,
			},
			"kubernetes": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				Description:   "Configuration block for the Kubernetes driver.",
				ConflictsWith: []string{"docker_container", "remote", "driver", "driver_options"},
				ForceNew:      true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the image to use for running BuildKit.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the Kubernetes namespace.",
						},
						"default_load": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Automatically load images to the Docker Engine image store. Defaults to `false`",
						},
						"replicas": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Sets the number of Pod replicas to create.",
						},
						"requests": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Resource requests for CPU, memory, and ephemeral storage.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "CPU limit for the Kubernetes pod.",
									},
									"memory": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Memory limit for the Kubernetes pod.",
									},
									"ephemeral_storage": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Ephemeral storage limit for the Kubernetes pod.",
									},
								},
							},
						},
						"limits": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Resource limits for CPU, memory, and ephemeral storage.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "CPU limit for the Kubernetes pod.",
									},
									"memory": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Memory limit for the Kubernetes pod.",
									},
									"ephemeral_storage": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Ephemeral storage limit for the Kubernetes pod.",
									},
								},
							},
						},
						"nodeselector": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the pod's nodeSelector label(s).",
						},
						"annotations": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets additional annotations on the deployments and pods.",
						},
						"labels": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets additional labels on the deployments and pods.",
						},
						"tolerations": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Configures the pod's taint toleration.",
						},
						"serviceaccount": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the pod's serviceAccountName.",
						},
						"schedulername": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the scheduler responsible for scheduling the pod.",
						},
						"timeout": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "120s",
							Description: "Set the timeout limit for pod provisioning.",
						},
						"rootless": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Run the container as a non-root user.",
						},
						"loadbalance": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "sticky",
							Description: "Load-balancing strategy (sticky or random).",
						},
						"qemu": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "QEMU emulation configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"install": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Install QEMU emulation for multi-platform support.",
									},
									"image": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "tonistiigi/binfmt:latest",
										Description: "Sets the QEMU emulation image.",
									},
								},
							},
						},
					},
				},
			},
			"docker_container": {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				MaxItems:      1,
				Description:   "Configuration block for the Docker-Container driver.",
				ConflictsWith: []string{"kubernetes", "remote", "driver", "driver_options"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the BuildKit image to use for the container.",
						},
						"memory": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the amount of memory the container can use.",
						},
						"memory_swap": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the memory swap limit for the container.",
						},
						"cpu_quota": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Imposes a CPU CFS quota on the container.",
						},
						"cpu_period": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the CPU CFS scheduler period for the container.",
						},
						"cpu_shares": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Configures CPU shares (relative weight) of the container.",
						},
						"cpuset_cpus": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Limits the set of CPU cores the container can use.",
						},
						"cpuset_mems": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Limits the set of CPU memory nodes the container can use.",
						},
						"default_load": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Automatically load images to the Docker Engine image store. Defaults to `false`",
						},
						"network": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the network mode for the container.",
						},
						"cgroup_parent": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "/docker/buildx",
							Description: "Sets the cgroup parent of the container if Docker is using the \"cgroupfs\" driver.",
						},
						"restart_policy": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "unless-stopped",
							Description: "Sets the container's restart policy.",
						},
						"env": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Sets environment variables in the container.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"remote": {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				MaxItems:      1,
				Description:   "Configuration block for the Remote driver.",
				ConflictsWith: []string{"kubernetes", "docker_container", "driver", "driver_options"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Sets the TLS client key.",
						},
						"cert": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Absolute path to the TLS client certificate to present to buildkitd.",
						},
						"cacert": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Absolute path to the TLS certificate authority used for validation.",
						},
						"servername": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "TLS server name used in requests.",
						},
						"default_load": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Automatically load images to the Docker Engine image store. Defaults to `false`",
						},
					},
				},
			},
		},
	}

}

// resourceDockerBuildxBuilderCreate handles the creation of a Buildx builder
func resourceDockerBuildxBuilderCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	name := d.Get("name").(string)
	driver := d.Get("driver").(string)
	platform := d.Get("platform").([]interface{})
	driverOptions := make([]string, 0)
	v, ok := d.GetOk("driver_options")
	if ok {
		driverOptions = processDriverOptions(v.(map[string]interface{}))
	}
	appendAction := d.Get("append").(bool)
	use := d.Get("use").(bool)

	log.Printf("[DEBUG] Creating Buildx builder: %s", name)

	if kubernetesConfig, ok := d.GetOk("kubernetes"); ok {
		driver = "kubernetes"
		kubernetes := kubernetesConfig.([]interface{})[0].(map[string]interface{})
		driverOptions = processDriverOptions(kubernetes)
	}

	if dockerContainerConfig, ok := d.GetOk("docker_container"); ok {
		driver = "docker-container"
		dockerContainer := dockerContainerConfig.([]interface{})[0].(map[string]interface{})
		driverOptions = processDriverOptions(dockerContainer)
	}

	// Updated the Create function to handle all Remote driver-specific parameters
	if remoteConfig, ok := d.GetOk("remote"); ok {
		driver = "remote"
		remote := remoteConfig.([]interface{})[0].(map[string]interface{})
		driverOptions = processDriverOptions(remote)
	}

	client := meta.(*ProviderConfig).DockerClient

	t, error := command.NewDockerCli()
	if error != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker CLI: %w", error))
	}

	log.Printf("[DEBUG] Docker CLI initialized %#v, %#v", client, client.DaemonHost())
	err := t.Initialize(&flags.ClientOptions{Hosts: []string{client.DaemonHost()}})

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to initialize Docker CLI: %w", err))
	}

	txn, release, err := storeutil.GetStore(t)

	if err != nil {
		return diag.FromErr(err)
	}
	// Ensure the file lock gets released no matter what happens.
	defer release()

	log.Printf("[DEBUG] Creating Buildx builder with name: %s", name)
	log.Printf("[DEBUG] Driver: %s", driver)
	log.Printf("[DEBUG] Driver options: %s", driverOptions)

	var ep string
	v = d.Get("endpoint").(string)
	if v != "" {
		ep = v.(string)
	}

	b, err := builder.Create(ctx, txn, t, builder.CreateOpts{
		Name:                name,
		Driver:              driver,
		NodeName:            d.Get("node").(string),
		Platforms:           stringListToStringSlice(platform),
		DriverOpts:          driverOptions,
		BuildkitdFlags:      d.Get("buildkit_flags").(string),
		BuildkitdConfigFile: d.Get("buildkit_config").(string),
		Use:                 use,
		Endpoint:            ep,
		Append:              appendAction,
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Buildx builder: %w", err))
	}

	// The store is no longer used from this point.
	// Release it so we aren't holding the file lock during the boot.
	release()

	if d.Get("bootstrap").(bool) {
		if _, err = b.Boot(ctx); err != nil {
			return diag.FromErr(fmt.Errorf("failed to bootstrap Buildx builder: %w", err))
		}
	}

	d.SetId(b.Name)
	d.Set("name", b.Name)

	return resourceDockerBuildxBuilderRead(ctx, d, meta)
}

func processDriverOptions(driverOptionsMap map[string]interface{}) []string {
	// Iterate over the driver options and append them to a string list
	resultStringList := make([]string, 0)

	for key, value := range driverOptionsMap {
		// replace underscores with dashes in the key
		key = strings.ReplaceAll(key, "_", "-")
		if strValue, ok := value.(string); ok {
			if strValue == "" {
				continue
			}
			resultStringList = append(resultStringList, fmt.Sprintf("%s=%s", key, strValue))
		} else if boolValue, ok := value.(bool); ok && boolValue {
			resultStringList = append(resultStringList, fmt.Sprintf("%s=true", key))
		}
		// handle TypeMap values
		if strMap, ok := value.(map[string]interface{}); ok {
			resultStringList = append(resultStringList, processDriverOptions(strMap)...)
		}
	}
	return resultStringList
}

// resourceDockerBuildxBuilderRead handles reading the state of a Buildx builder
// corresponding file in buildx repo: https://github.com/docker/buildx/blob/master/commands/inspect.go
func resourceDockerBuildxBuilderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	dockerCli, error := command.NewDockerCli()
	if error != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker CLI: %w", error))
	}
	err := dockerCli.Initialize(&flags.ClientOptions{Hosts: []string{client.DaemonHost()}})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to initialize Docker CLI: %w", err))
	}
	name := d.Id()

	log.Printf("[DEBUG] Reading Buildx builder: %s", name)

	_, err = builder.New(dockerCli,
		builder.WithName(d.Get("name").(string)),
		builder.WithSkippedValidation(),
	)
	if err != nil {
		log.Printf("[DEBUG] Failed to read Buildx builder %s: %v", name, err)
		if strings.Contains(err.Error(), fmt.Sprintf("no builder \"%s\" found", name)) {
			log.Printf("[DEBUG] Buildx builder %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

// resourceDockerBuildxBuilderDelete handles the deletion of a Buildx builder
// corresponding file in buildx repo: https://github.com/docker/buildx/blob/master/commands/rm.go
func resourceDockerBuildxBuilderDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Id()
	log.Printf("[DEBUG] Deleting Buildx builder: %s", name)

	client := meta.(*ProviderConfig).DockerClient

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker CLI: %w", err))
	}
	err = dockerCli.Initialize(&flags.ClientOptions{Hosts: []string{client.DaemonHost()}})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to initialize Docker CLI: %w", err))
	}

	txn, release, err := storeutil.GetStore(dockerCli)

	if err != nil {
		return diag.FromErr(err)
	}

	defer release()

	eg, _ := errgroup.WithContext(ctx)
	func(name string) {
		eg.Go(func() (err error) {
			defer func() {
				if err == nil {
					_, _ = fmt.Fprintf(dockerCli.Err(), "%s removed\n", name)
				} else {
					_, _ = fmt.Fprintf(dockerCli.Err(), "failed to remove %s: %v\n", name, err)
				}
			}()

			b, err := builder.New(dockerCli,
				builder.WithName(name),
				builder.WithStore(txn),
				builder.WithSkippedValidation(),
			)
			if err != nil {
				return err
			}

			nodes, err := b.LoadNodes(ctx)
			if err != nil {
				return err
			}

			if cb := b.ContextName(); cb != "" {
				return errors.Errorf("context builder cannot be removed, run `docker context rm %s` to remove this context", cb)
			}

			err1 := rm(ctx, nodes, rmOptions{keepState: false, keepDaemon: false, allInactive: false, force: false})
			if err := txn.Remove(b.Name); err != nil {
				return err
			}
			if err1 != nil {
				return err1
			}

			return nil
		})
	}(name)

	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to remove one or more builders")
	}
	return nil
}

func rm(ctx context.Context, nodes []builder.Node, in rmOptions) (err error) {
	for _, node := range nodes {
		if node.Driver == nil {
			continue
		}
		// Do not stop the buildkitd daemon when --keep-daemon is provided
		if !in.keepDaemon {
			if err := node.Driver.Stop(ctx, true); err != nil {
				return err
			}
		}
		if err := node.Driver.Rm(ctx, true, !in.keepState, !in.keepDaemon); err != nil {
			return err
		}
		if node.Err != nil {
			err = node.Err
		}
	}
	return err
}

type rmOptions struct {
	keepState   bool
	keepDaemon  bool
	allInactive bool
	force       bool
}
