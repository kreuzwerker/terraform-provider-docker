package provider

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

	createOpts := volume.VolumeCreateBody{}

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

	var err error
	var retVolume types.Volume
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
		return diag.Errorf("Unable to inspect volume: %s", err)
	}

	jsonObj, _ := json.MarshalIndent(volume, "", "\t")
	log.Printf("[DEBUG] Docker volume inspect from readFunc: %s", jsonObj)

	d.Set("name", volume.Name)
	d.Set("labels", mapToLabelSet(volume.Labels))
	d.Set("driver", volume.Driver)
	d.Set("driver_opts", volume.Options)
	d.Set("mountpoint", volume.Mountpoint)

	return nil
}

func resourceDockerVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for volume: '%s' to get removed: max '%v seconds'", d.Id(), volumeReadRefreshTimeout)

	stateConf := &resource.StateChangeConf{
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

func resourceDockerVolumeRemoveRefreshFunc(
	volumeID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		forceDelete := true

		if err := client.VolumeRemove(context.Background(), volumeID, forceDelete); err != nil {
			if containsIgnorableErrorMessage(err.Error(), "volume is in use") {
				log.Printf("[INFO] Volume with id '%v' is still in use", volumeID)
				return volumeID, "in_use", nil
			}
			log.Printf("[INFO] Removing volume with id '%v' caused an error: %v", volumeID, err)
			return nil, "", err
		}
		log.Printf("[INFO] Removing volume with id '%v' got removed", volumeID)
		return volumeID, "removed", nil
	}
}
