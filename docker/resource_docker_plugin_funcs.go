package docker

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerPluginCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginName := d.Get("name").(string)
	// TODO
	body, err := client.PluginInstall(ctx, pluginName, types.PluginInstallOptions{
		RemoteRef:            pluginName,
		AcceptAllPermissions: true,
		Disabled:             d.Get("disabled").(bool),
	})
	if err != nil {
		return fmt.Errorf("install a Docker plugin "+pluginName+": %w", err)
	}
	_, _ = ioutil.ReadAll(body)
	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginName)
	if err != nil {
		return fmt.Errorf("inspect a Docker plugin "+pluginName+": %w", err)
	}
	setDockerPlugin(d, plugin)
	return nil
}

func setDockerPlugin(d *schema.ResourceData, plugin *types.Plugin) {
	d.SetId(plugin.ID)
	d.Set("name", plugin.Name)
	d.Set("disabled", !plugin.Enabled)
}

func resourceDockerPluginRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("inspect a Docker plugin "+pluginID+": %w", err)
	}
	setDockerPlugin(d, plugin)
	// TODO set values
	return nil
}

func resourceDockerPluginUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	// TODO
	if d.HasChange("disabled") {
		if d.Get("disabled").(bool) {
			if err := client.PluginDisable(ctx, pluginID, types.PluginDisableOptions{}); err != nil {
				return fmt.Errorf("disable the Docker plugin "+pluginID+": %w", err)
			}
		} else {
			if err := client.PluginEnable(ctx, pluginID, types.PluginEnableOptions{}); err != nil {
				return fmt.Errorf("enable the Docker plugin "+pluginID+": %w", err)
			}
		}
	}
	// if err := client.PluginSet(ctx, pluginID, nil); err != nil {
	// 	return fmt.Errorf("modifiy settings for the Docker plugin "+pluginID+": %w", err)
	// }
	// call the read function to update the resource's state.
	// https://learn.hashicorp.com/tutorials/terraform/provider-update?in=terraform/providers#implement-update
	return resourceDockerPluginRead(d, meta)
}

func resourceDockerPluginDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	destroyOptions := d.Get("destroy_option").([]interface{})
	force := false
	if len(destroyOptions) == 1 {
		destroyOption := destroyOptions[0].(map[string]interface{})
		f, ok := destroyOption["force"]
		if ok {
			force = f.(bool)
		}
	}
	if err := client.PluginRemove(ctx, pluginID, types.PluginRemoveOptions{
		Force: force,
	}); err != nil {
		return fmt.Errorf("remove the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}
