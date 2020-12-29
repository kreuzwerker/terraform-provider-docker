package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func getDockerPluginArgs(src interface{}) []string {
	if src == nil {
		return nil
	}
	b := src.(*schema.Set)
	args := make([]string, b.Len())
	for i, a := range b.List() {
		args[i] = a.(string)
	}
	return args
}

func resourceDockerPluginCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginRef := d.Get("plugin_reference").(string)
	alias := d.Get("alias").(string)
	log.Printf("[DEBUG] Install a Docker plugin " + pluginRef)
	body, err := client.PluginInstall(ctx, alias, types.PluginInstallOptions{
		RemoteRef:            pluginRef,
		AcceptAllPermissions: d.Get("grant_all_permissions").(bool),
		Disabled:             d.Get("disabled").(bool),
		Args:                 getDockerPluginArgs(d.Get("args")),
	})
	if err != nil {
		return fmt.Errorf("install a Docker plugin "+pluginRef+": %w", err)
	}
	_, _ = ioutil.ReadAll(body)
	key := pluginRef
	if alias != "" {
		key = alias
	}
	plugin, _, err := client.PluginInspectWithRaw(ctx, key)
	if err != nil {
		return fmt.Errorf("inspect a Docker plugin "+key+": %w", err)
	}
	setDockerPlugin(d, plugin)
	return nil
}

func setDockerPlugin(d *schema.ResourceData, plugin *types.Plugin) {
	d.SetId(plugin.ID)
	d.Set("plugin_reference", plugin.PluginReference)
	d.Set("alias", plugin.Name)
	d.Set("disabled", !plugin.Enabled)
	// TODO support other settings
	// https://docs.docker.com/engine/reference/commandline/plugin_set/#extended-description
	// source of mounts .Settings.Mounts
	// path of devices .Settings.Devices
	// args .Settings.Args
	d.Set("args", plugin.Settings.Env)
}

func resourceDockerPluginRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginID)
	if err != nil {
		log.Printf("[DEBUG] Inspect a Docker plugin "+pluginID+": %w", err)
		d.SetId("")
		return nil
	}
	setDockerPlugin(d, plugin)
	return nil
}

func setPluginArgs(ctx context.Context, d *schema.ResourceData, client *client.Client) error {
	if !d.HasChange("args") {
		return nil
	}
	pluginID := d.Id()
	log.Printf("[DEBUG] Update settings of a Docker plugin " + pluginID)
	if err := client.PluginSet(ctx, pluginID, getDockerPluginArgs(d.Get("args"))); err != nil {
		return fmt.Errorf("modifiy settings for the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}

func resourceDockerPluginUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	skipArgs := false
	if d.HasChange("disabled") {
		if d.Get("disabled").(bool) {
			log.Printf("[DEBUG] Disable a Docker plugin " + pluginID)
			if err := client.PluginDisable(ctx, pluginID, types.PluginDisableOptions{
				Force: d.Get("force_disable").(bool),
			}); err != nil {
				return fmt.Errorf("disable the Docker plugin "+pluginID+": %w", err)
			}
		} else {
			if err := setPluginArgs(ctx, d, client); err != nil {
				return err
			}
			skipArgs = true
			log.Printf("[DEBUG] Enable a Docker plugin " + pluginID)
			if err := client.PluginEnable(ctx, pluginID, types.PluginEnableOptions{
				Timeout: d.Get("enable_timeout").(int),
			}); err != nil {
				return fmt.Errorf("enable the Docker plugin "+pluginID+": %w", err)
			}
		}
	}
	if !skipArgs {
		plugin, _, err := client.PluginInspectWithRaw(ctx, pluginID)
		if err != nil {
			return fmt.Errorf("inspect a Docker plugin "+pluginID+": %w", err)
		}
		f := false
		if plugin.Enabled && d.Get("disable_when_set").(bool) {
			// temporarily disable the plugin before updating the plugin settings
			log.Printf("[DEBUG] Disable a Docker plugin " + pluginID + " temporarily before updating teh pugin settings")
			if err := client.PluginDisable(ctx, pluginID, types.PluginDisableOptions{
				Force: d.Get("force_disable").(bool),
			}); err != nil {
				return fmt.Errorf("disable the Docker plugin "+pluginID+": %w", err)
			}
			f = true
		}
		if err := setPluginArgs(ctx, d, client); err != nil {
			return err
		}
		if f {
			log.Printf("[DEBUG] Enable a Docker plugin " + pluginID)
			if err := client.PluginEnable(ctx, pluginID, types.PluginEnableOptions{
				Timeout: d.Get("enable_timeout").(int),
			}); err != nil {
				return fmt.Errorf("enable the Docker plugin "+pluginID+": %w", err)
			}
		}
	}
	// call the read function to update the resource's state.
	// https://learn.hashicorp.com/tutorials/terraform/provider-update?in=terraform/providers#implement-update
	return resourceDockerPluginRead(d, meta)
}

func resourceDockerPluginDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginID := d.Id()
	log.Printf("[DEBUG] Remove a Docker plugin " + pluginID)
	if err := client.PluginRemove(ctx, pluginID, types.PluginRemoveOptions{
		Force: d.Get("force_destroy").(bool),
	}); err != nil {
		return fmt.Errorf("remove the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}
