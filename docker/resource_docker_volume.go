package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerVolumeCreate,
		Read:   resourceDockerVolumeRead,
		Delete: resourceDockerVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"labels": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     labelSchema,
			},
			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"mountpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerVolumeV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return replaceLabelsMapFieldWithSetField(rawState), nil
				},
			},
		},
	}
}

func resourceDockerVolumeV0() *schema.Resource {
	return &schema.Resource{
		// This is only used for state migration, so the CRUD
		// callbacks are no longer relevant
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"mountpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDockerVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()

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
		return fmt.Errorf("Unable to create volume: %s", err)
	}

	d.SetId(retVolume.Name)
	return resourceDockerVolumeRead(d, meta)
}

func resourceDockerVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()

	var err error
	var retVolume types.Volume
	retVolume, err = client.VolumeInspect(ctx, d.Id())

	if err != nil {
		return fmt.Errorf("Unable to inspect volume: %s", err)
	}

	d.Set("name", retVolume.Name)
	d.Set("labels", mapToLabelSet(retVolume.Labels))
	d.Set("driver", retVolume.Driver)
	d.Set("driver_opts", retVolume.Options)
	d.Set("mountpoint", retVolume.Mountpoint)

	return nil
}

func resourceDockerVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Waiting for volume: '%s' to get removed: max '%v seconds'", d.Id(), 30)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"in_use"},
		Target:     []string{"removed"},
		Refresh:    resourceDockerVolumeRemoveRefreshFunc(d.Id(), meta),
		Timeout:    30 * time.Second,
		MinTimeout: 5 * time.Second,
		Delay:      2 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
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
			if strings.Contains(err.Error(), "volume is in use") { // store.IsInUse(err)
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
