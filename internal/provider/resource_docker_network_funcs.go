package provider

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	networkReadRefreshTimeout               = 30 * time.Second
	networkReadRefreshWaitBeforeRefreshes   = 5 * time.Second
	networkReadRefreshDelay                 = 2 * time.Second
	networkRemoveRefreshTimeout             = 30 * time.Second
	networkRemoveRefreshWaitBeforeRefreshes = 5 * time.Second
	networkRemoveRefreshDelay               = 2 * time.Second
)

func resourceDockerNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	createOpts := types.NetworkCreate{}
	if v, ok := d.GetOk("labels"); ok {
		createOpts.Labels = labelSetToMap(v.(*schema.Set))
	}
	if v, ok := d.GetOk("check_duplicate"); ok {
		createOpts.CheckDuplicate = v.(bool)
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("options"); ok {
		createOpts.Options = mapTypeMapValsToString(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("internal"); ok {
		createOpts.Internal = v.(bool)
	}
	if v, ok := d.GetOk("attachable"); ok {
		createOpts.Attachable = v.(bool)
	}
	if v, ok := d.GetOk("ingress"); ok {
		createOpts.Ingress = v.(bool)
	}
	if v, ok := d.GetOk("ipv6"); ok {
		createOpts.EnableIPv6 = v.(bool)
	}

	ipamOpts := &network.IPAM{}
	ipamOptsSet := false
	if v, ok := d.GetOk("ipam_driver"); ok {
		ipamOpts.Driver = v.(string)
		ipamOptsSet = true
	}
	if v, ok := d.GetOk("ipam_config"); ok {
		ipamOpts.Config = ipamConfigSetToIpamConfigs(v.(*schema.Set))
		ipamOptsSet = true
	}
	if v, ok := d.GetOk("ipam_options"); ok {
		ipamOpts.Options = mapTypeMapValsToString(v.(map[string]interface{}))
		ipamOptsSet = true
	}

	if ipamOptsSet {
		createOpts.IPAM = ipamOpts
	}

	retNetwork, err := client.NetworkCreate(ctx, d.Get("name").(string), createOpts)
	if err != nil {
		return diag.Errorf("Unable to create network: %s", err)
	}

	d.SetId(retNetwork.ID)
	// d.Set("check_duplicate") TODO mavogel
	return resourceDockerNetworkRead(ctx, d, meta)
}

func resourceDockerNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for network: '%s' to expose all fields: max '%v seconds'", d.Id(), networkReadRefreshTimeout)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"all_fields", "removed"},
		Refresh:    resourceDockerNetworkReadRefreshFunc(ctx, d, meta),
		Timeout:    networkReadRefreshTimeout,
		MinTimeout: networkReadRefreshWaitBeforeRefreshes,
		Delay:      networkReadRefreshDelay,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDockerNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for network: '%s' to be removed: max '%v seconds'", d.Id(), networkRemoveRefreshTimeout)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"removed"},
		Refresh:    resourceDockerNetworkRemoveRefreshFunc(ctx, d, meta),
		Timeout:    networkRemoveRefreshTimeout,
		MinTimeout: networkRemoveRefreshWaitBeforeRefreshes,
		Delay:      networkRemoveRefreshDelay,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func ipamConfigSetToIpamConfigs(ipamConfigSet *schema.Set) []network.IPAMConfig {
	ipamConfigs := make([]network.IPAMConfig, ipamConfigSet.Len())

	for i, ipamConfigInt := range ipamConfigSet.List() {
		ipamConfigRaw := ipamConfigInt.(map[string]interface{})

		ipamConfig := network.IPAMConfig{}
		ipamConfig.Subnet = ipamConfigRaw["subnet"].(string)
		ipamConfig.IPRange = ipamConfigRaw["ip_range"].(string)
		ipamConfig.Gateway = ipamConfigRaw["gateway"].(string)

		auxAddressRaw := ipamConfigRaw["aux_address"].(map[string]interface{})
		ipamConfig.AuxAddress = make(map[string]string, len(auxAddressRaw))
		for k, v := range auxAddressRaw {
			ipamConfig.AuxAddress[k] = v.(string)
		}

		ipamConfigs[i] = ipamConfig
	}

	return ipamConfigs
}

func resourceDockerNetworkReadRefreshFunc(ctx context.Context,
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		networkID := d.Id()

		retNetwork, _, err := client.NetworkInspectWithRaw(ctx, networkID, types.NetworkInspectOptions{})
		if err != nil {
			log.Printf("[WARN] Network (%s) not found, removing from state", networkID)
			d.SetId("")
			return networkID, "removed", nil
		}

		jsonObj, _ := json.MarshalIndent(retNetwork, "", "\t")
		log.Printf("[DEBUG] Docker network inspect: %s", jsonObj)

		d.Set("name", retNetwork.Name)
		d.Set("labels", mapToLabelSet(retNetwork.Labels))
		d.Set("driver", retNetwork.Driver)
		d.Set("internal", retNetwork.Internal)
		d.Set("attachable", retNetwork.Attachable)
		d.Set("ingress", retNetwork.Ingress)
		d.Set("ipv6", retNetwork.EnableIPv6)
		d.Set("ipam_driver", retNetwork.IPAM.Driver)
		d.Set("ipam_options", retNetwork.IPAM.Options)
		d.Set("scope", retNetwork.Scope)
		if retNetwork.Scope == "overlay" {
			if retNetwork.Options != nil && len(retNetwork.Options) != 0 {
				d.Set("options", retNetwork.Options)
			} else {
				log.Printf("[DEBUG] options: %v not exposed", retNetwork.Options)
				return networkID, "pending", nil
			}
		} else {
			d.Set("options", retNetwork.Options)
		}

		if err = d.Set("ipam_config", flattenIpamConfigSpec(retNetwork.IPAM.Config)); err != nil {
			log.Printf("[WARN] failed to set ipam config from API: %s", err)
		}

		log.Println("[DEBUG] all network fields exposed")
		return networkID, "all_fields", nil
	}
}

func resourceDockerNetworkRemoveRefreshFunc(ctx context.Context,
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		networkID := d.Id()

		_, _, err := client.NetworkInspectWithRaw(ctx, networkID, types.NetworkInspectOptions{})
		if err != nil {
			log.Printf("[INFO] Network (%s) not found. Already removed", networkID)
			return networkID, "removed", nil
		}

		if err := client.NetworkRemove(ctx, networkID); err != nil {
			if containsIgnorableErrorMessage(err.Error(), "has active endpoints") {
				return networkID, "pending", nil
			}
			return networkID, "other", err
		}

		return networkID, "removed", nil
	}
}
