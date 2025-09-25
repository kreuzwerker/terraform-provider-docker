package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	volumeReadRefreshTimeout             = 30 * time.Second
	volumeReadRefreshWaitBeforeRefreshes = 5 * time.Second
	volumeReadRefreshDelay               = 2 * time.Second
)

func resourceDockerVolume() *schema.Resource {
	return &schema.Resource{
		Description: "Creates and destroys a volume in Docker. This can be used alongside [docker_container](container.md) to prepare volumes that can be shared across containers.",

		CreateContext: resourceDockerVolumeCreate,
		ReadContext:   resourceDockerVolumeRead,
		DeleteContext: resourceDockerVolumeDelete,
		UpdateContext: resourceDockerVolumeUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker volume (will be generated if not provided).",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
				Elem:        labelSchema,
			},
			"driver": {
				Type:        schema.TypeString,
				Description: "Driver type for the volume. Defaults to `local`.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"driver_opts": {
				Type:        schema.TypeMap,
				Description: "Options specific to the driver.",
				Optional:    true,
				ForceNew:    true,
			},
			"mountpoint": {
				Type:        schema.TypeString,
				Description: "The mountpoint of the volume.",
				Computed:    true,
			},
			"cluster": {
				Type:        schema.TypeList,
				Description: "Cluster-specific options for volume creation. Only works if the Docker daemon is running in swarm mode and is the swarm manager.",
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "The ID of the cluster volume.",
							Computed:    true,
						},
						"scope": {
							Type:         schema.TypeString,
							Description:  "The scope of the volume. Can be `single` (default) or `multi`.",
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"single", "multi"}, false),
							Default:      "single",
						},
						"sharing": {
							Type:         schema.TypeString,
							Description:  "The sharing mode. Can be `none` (default), `readonly`, `onewriter` or `all`.",
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"none", "readonly", "onewriter", "all"}, false),
							Default:      "none",
						},
						"type": {
							Type:         schema.TypeString,
							Description:  "Cluster Volume access type. Can be `mount` or `block` (default).",
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"mount", "block"}, false),
							Default:      "block",
						},
						"topology_preferred": {
							Type:        schema.TypeString,
							Description: "A topology that the Cluster Volume would be preferred in",
							Optional:    true,
							ForceNew:    true,
							Default:     "",
						},
						"topology_required": {
							Type:        schema.TypeString,
							Description: "A topology that the Cluster Volume must be accessible from",
							Optional:    true,
							ForceNew:    true,
							Default:     "",
						},
						"required_bytes": {
							Type:        schema.TypeString,
							Description: "Maximum size of the Cluster Volume in human readable memory bytes (like 128MiB, 2GiB, etc). Must be in format of KiB, MiB, Gib, Tib or PiB.",
							Optional:    true,
							ForceNew:    true,
						},
						"limit_bytes": {
							Type:        schema.TypeString,
							Description: "Minimum size of the Cluster Volume in human readable memory bytes (like 128MiB, 2GiB, etc). Must be in format of KiB, MiB, Gib, Tib or PiB.",
							Optional:    true,
							ForceNew:    true,
						},
						"group": {
							Type:        schema.TypeString,
							Description: "Cluster Volume group",
							Optional:    true,
							ForceNew:    true,
							Default:     "",
						},
						"availability": {
							Type:         schema.TypeString,
							Description:  "Availability of the volume. Can be `active` (default), `pause`, or `drain`.",
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"active", "pause", "drain"}, false),
							Default:      "active",
						},
					},
				},
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerVolumeV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return replaceLabelsMapFieldWithSetField(rawState), nil
				},
			},
		},
	}
}

func resourceDockerVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	createOpts := volume.CreateOptions{}

	if v, ok := d.GetOk("name"); ok {
		createOpts.Name = v.(string)
	}
	if v, ok := d.GetOk("labels"); ok {
		createOpts.Labels = labelSetToMap(v.(*schema.Set))
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("driver_opts"); ok {
		createOpts.DriverOpts = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	// Handle cluster volume options
	if v, ok := d.GetOk("cluster"); ok {
		if client.ClientVersion() < "1.42" {
			return diag.Errorf("You have supplied a 'cluster' block for your docker_volume resource. This is supported starting with client version '1.42', but you have %s", client.ClientVersion())
		}

		clusterList := v.([]interface{})
		if len(clusterList) > 0 {
			clusterConfig := clusterList[0].(map[string]interface{})

			createOpts.ClusterVolumeSpec = &volume.ClusterVolumeSpec{
				Group: clusterConfig["group"].(string),
				AccessMode: &volume.AccessMode{
					Scope:   volume.Scope(clusterConfig["scope"].(string)),
					Sharing: volume.SharingMode(clusterConfig["sharing"].(string)),
				},
				Availability: volume.Availability(clusterConfig["availability"].(string)),
			}

			switch clusterConfig["type"].(string) {
			case "mount":
				createOpts.ClusterVolumeSpec.AccessMode.MountVolume = &volume.TypeMount{}
			case "block":
				createOpts.ClusterVolumeSpec.AccessMode.BlockVolume = &volume.TypeBlock{}
			}

			vcr := &volume.CapacityRange{}
			var memBytes opts.MemBytes
			if r := clusterConfig["required_bytes"].(string); len(r) > 0 {
				if err := memBytes.Set(r); err != nil {
					return diag.Errorf("Invalid value for required_bytes: %s", err)
				}
				vcr.RequiredBytes = memBytes.Value()
				fmt.Printf("[DEBUG] Required bytes set to %d\n", vcr.RequiredBytes)
			}

			if l := clusterConfig["limit_bytes"].(string); len(l) > 0 {
				if err := memBytes.Set(l); err != nil {
					return diag.Errorf("Invalid value for limit_bytes: %s", err)
				}
				vcr.LimitBytes = memBytes.Value()
				fmt.Printf("[DEBUG] Limit bytes set to %d\n", vcr.LimitBytes)
			}
			createOpts.ClusterVolumeSpec.CapacityRange = vcr

			if secretsList, ok := clusterConfig["secrets"].([]interface{}); ok && len(secretsList) > 0 {
				secrets := make([]volume.Secret, len(secretsList))
				for i, secretItem := range secretsList {
					secret := secretItem.(map[string]interface{})
					secrets[i] = volume.Secret{
						Key:    secret["key"].(string),
						Secret: secret["secret"].(string),
					}
				}
				createOpts.ClusterVolumeSpec.Secrets = secrets
				sort.SliceStable(createOpts.ClusterVolumeSpec.Secrets, func(i, j int) bool {
					return createOpts.ClusterVolumeSpec.Secrets[i].Key < createOpts.ClusterVolumeSpec.Secrets[j].Key
				})
			}

			topology := &volume.TopologyRequirement{}

			if topology_required, ok := clusterConfig["topology_required"].(string); ok && len(topology_required) > 0 {
				segments := map[string]string{}
				for _, segment := range strings.Split(topology_required, ",") {
					k, v, _ := strings.Cut(segment, "=")
					segments[k] = v
				}

				topology.Requisite = append(
					topology.Requisite,
					volume.Topology{Segments: segments},
				)
			}
			if topology_preferred, ok := clusterConfig["topology_preferred"].(string); ok && len(topology_preferred) > 0 {
				segments := map[string]string{}
				for _, segment := range strings.Split(topology_preferred, ",") {
					k, v, _ := strings.Cut(segment, "=")
					segments[k] = v
				}

				topology.Preferred = append(
					topology.Preferred,
					volume.Topology{Segments: segments},
				)
			}

			createOpts.ClusterVolumeSpec.AccessibilityRequirements = topology
		}
	}
	var err error
	var retVolume volume.Volume
	retVolume, err = client.VolumeCreate(ctx, createOpts)

	if err != nil {
		return diag.Errorf("Unable to create volume: %s", err)
	}

	d.SetId(retVolume.Name)
	return resourceDockerVolumeRead(ctx, d, meta)
}

func resourceDockerVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	volume, err := client.VolumeInspect(ctx, d.Id())

	if err != nil {
		if errdefs.IsNotFound(err) {
			log.Printf("[WARN] Volume with id `%s` not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("Unable to inspect volume: %s", err)
	}

	jsonObj, _ := json.MarshalIndent(volume, "", "\t")
	log.Printf("[DEBUG] Docker volume inspect from readFunc: %s", jsonObj)

	d.Set("name", volume.Name)
	d.Set("labels", mapToLabelSet(volume.Labels))
	d.Set("driver", volume.Driver)
	d.Set("driver_opts", volume.Options)
	d.Set("mountpoint", volume.Mountpoint)

	// check if volume.ClusterVolume is set
	if volume.ClusterVolume != nil {

		typeValue := "block"
		if volume.ClusterVolume.Spec.AccessMode.MountVolume != nil {
			typeValue = "mount"
		}

		// loop over volume.ClusterVolume.Spec.AccessibilityRequirements.Preferred
		topologyPreferred := []string{}
		for _, segment := range volume.ClusterVolume.Spec.AccessibilityRequirements.Preferred {
			for key, value := range segment.Segments {
				topologyPreferred = append(topologyPreferred, fmt.Sprintf("%s=%s", key, value))
			}
		}
		topologyRequired := []string{}
		for _, segment := range volume.ClusterVolume.Spec.AccessibilityRequirements.Requisite {
			for key, value := range segment.Segments {
				topologyRequired = append(topologyRequired, fmt.Sprintf("%s=%s", key, value))
			}
		}

		d.Set("cluster", []interface{}{
			map[string]interface{}{
				"id":           volume.ClusterVolume.ID,
				"scope":        volume.ClusterVolume.Spec.AccessMode.Scope,
				"sharing":      volume.ClusterVolume.Spec.AccessMode.Sharing,
				"group":        volume.ClusterVolume.Spec.Group,
				"availability": volume.ClusterVolume.Spec.Availability,
				"type":         typeValue,
				"required_bytes": func() string {
					mb := opts.MemBytes(0)
					mb = opts.MemBytes(volume.ClusterVolume.Spec.CapacityRange.RequiredBytes)
					return mb.String()
				}(),
				"limit_bytes": func() string {
					mb := opts.MemBytes(0)
					mb = opts.MemBytes(volume.ClusterVolume.Spec.CapacityRange.LimitBytes)
					return mb.String()
				}(),
				"topology_preferred": strings.Join(topologyPreferred, ","),
				"topology_required":  strings.Join(topologyRequired, ","),
			},
		})
	}
	return nil
}

func resourceDockerVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for volume: `%s` to get removed: max `%v seconds`", d.Id(), volumeReadRefreshTimeout)

	stateConf := &retry.StateChangeConf{
		Pending:    []string{"in_use"},
		Target:     []string{"removed"},
		Refresh:    resourceDockerVolumeRemoveRefreshFunc(d.Id(), meta),
		Timeout:    volumeReadRefreshTimeout,
		MinTimeout: volumeReadRefreshWaitBeforeRefreshes,
		Delay:      volumeReadRefreshDelay,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceDockerVolumeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	attrs := []string{
		"cluster.availability",
	}
	for _, attr := range attrs {
		if d.HasChange(attr) {
			clusterList := d.Get("cluster").([]interface{})
			if len(clusterList) > 0 {
				clusterConfig := clusterList[0].(map[string]interface{})
				client := meta.(*ProviderConfig).DockerClient

				vol, _, err := client.VolumeInspectWithRaw(ctx, clusterConfig["id"].(string))
				if err != nil {
					return diag.FromErr(err)
				}

				if vol.ClusterVolume == nil || d.Get("cluster") == nil {
					return diag.Errorf("Can only update cluster volumes")
				}

				vol.ClusterVolume.Spec.Availability = volume.Availability(clusterConfig["availability"].(string))

				err = client.VolumeUpdate(
					ctx, vol.ClusterVolume.ID, vol.ClusterVolume.Version,
					volume.UpdateOptions{
						Spec: &vol.ClusterVolume.Spec,
					},
				)
				if err != nil {
					return diag.Errorf("Unable to update the cluster volume: %v", err)
				}
				break
			}
		}
	}
	return nil
}

func resourceDockerVolumeRemoveRefreshFunc(
	volumeID string, meta interface{}) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		forceDelete := true

		if err := client.VolumeRemove(context.Background(), volumeID, forceDelete); err != nil {
			if containsIgnorableErrorMessage(err.Error(), "volume is in use") {
				log.Printf("[INFO] Volume with id `%v` is still in use", volumeID)
				return volumeID, "in_use", nil
			}
			log.Printf("[INFO] Removing volume with id `%v` caused an error: %v", volumeID, err)
			return nil, "", err
		}
		log.Printf("[INFO] Removing volume with id `%v` got removed", volumeID)
		return volumeID, "removed", nil
	}
}
