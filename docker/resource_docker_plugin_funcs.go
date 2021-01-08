package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func getDockerPluginEnv(src interface{}) []string {
	if src == nil {
		return nil
	}
	b := src.(*schema.Set)
	envs := make([]string, b.Len())
	for i, a := range b.List() {
		envs[i] = a.(string)
	}
	return envs
}

func dockerPluginGrantPermissionsSetFunc(v interface{}) int {
	return schema.HashString(v.(map[string]interface{})["name"].(string))
}

func complementTag(image string) string {
	if strings.Contains(image, ":") {
		return image
	}
	return image + ":latest"
}

func getDockerPluginGrantPermissions(src interface{}) func(types.PluginPrivileges) (bool, error) {
	grantPermissionsSet := src.(*schema.Set)
	grantPermissions := make(map[string]map[string]struct{}, grantPermissionsSet.Len())
	for _, b := range grantPermissionsSet.List() {
		c := b.(map[string]interface{})
		name := c["name"].(string)
		values := c["value"].(*schema.Set)
		grantPermission := make(map[string]struct{}, values.Len())
		for _, value := range values.List() {
			grantPermission[value.(string)] = struct{}{}
		}
		grantPermissions[name] = grantPermission
	}
	return func(privileges types.PluginPrivileges) (bool, error) {
		for _, privilege := range privileges {
			grantPermission, nameOK := grantPermissions[privilege.Name]
			if !nameOK {
				log.Print("[DEBUG] to install the plugin, the following permissions are required: " + privilege.Name)
				return false, nil
			}
			for _, value := range privilege.Value {
				if _, ok := grantPermission[value]; !ok {
					log.Print("[DEBUG] to install the plugin, the following permissions are required: " + privilege.Name + " " + value)
					return false, nil
				}
			}
		}
		return true, nil
	}
}

func resourceDockerPluginCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	ctx := context.Background()
	pluginRef := d.Get("plugin_reference").(string)
	alias := d.Get("alias").(string)
	log.Printf("[DEBUG] Install a Docker plugin " + pluginRef)
	opts := types.PluginInstallOptions{
		RemoteRef:            pluginRef,
		AcceptAllPermissions: d.Get("grant_all_permissions").(bool),
		Disabled:             !d.Get("enabled").(bool),
		// TODO support other settings
		Args: getDockerPluginEnv(d.Get("env")),
	}
	if v, ok := d.GetOk("grant_permissions"); ok {
		opts.AcceptPermissionsFunc = getDockerPluginGrantPermissions(v)
	}
	body, err := client.PluginInstall(ctx, alias, opts)
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
	d.Set("enabled", plugin.Enabled)
	// TODO support other settings
	// https://docs.docker.com/engine/reference/commandline/plugin_set/#extended-description
	// source of mounts .Settings.Mounts
	// path of devices .Settings.Devices
	// args .Settings.Args
	d.Set("env", plugin.Settings.Env)
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

func disablePlugin(ctx context.Context, d *schema.ResourceData, cl *client.Client) error {
	pluginID := d.Id()
	log.Printf("[DEBUG] Disable a Docker plugin " + pluginID)
	if err := cl.PluginDisable(ctx, pluginID, types.PluginDisableOptions{
		Force: d.Get("force_disable").(bool),
	}); err != nil {
		return fmt.Errorf("disable the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}

func enablePlugin(ctx context.Context, d *schema.ResourceData, cl *client.Client) error {
	pluginID := d.Id()
	log.Print("[DEBUG] Enable a Docker plugin " + pluginID)
	if err := cl.PluginEnable(ctx, pluginID, types.PluginEnableOptions{
		Timeout: d.Get("enable_timeout").(int),
	}); err != nil {
		return fmt.Errorf("enable the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}

func pluginSet(ctx context.Context, d *schema.ResourceData, cl *client.Client) error {
	pluginID := d.Id()
	log.Printf("[DEBUG] Update settings of a Docker plugin " + pluginID)
	// currently, only environment variables are supported.
	// TODO support other args
	// https://docs.docker.com/engine/reference/commandline/plugin_set/#extended-description
	// source of mounts .Settings.Mounts
	// path of devices .Settings.Devices
	// args .Settings.Args
	if err := cl.PluginSet(ctx, pluginID, getDockerPluginEnv(d.Get("env"))); err != nil {
		return fmt.Errorf("modifiy settings for the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}

func pluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) (gErr error) {
	cl := meta.(*ProviderConfig).DockerClient
	o, n := d.GetChange("enabled")
	oldEnabled, newEnabled := o.(bool), n.(bool)
	if d.HasChange("env") {
		if oldEnabled {
			// To update the plugin setttings, the plugin must be disabled
			if err := disablePlugin(ctx, d, cl); err != nil {
				return err
			}
			if newEnabled {
				defer func() {
					if err := enablePlugin(ctx, d, cl); err != nil {
						if gErr == nil {
							gErr = err
							return
						}
					}
				}()
			}
		}
		if err := pluginSet(ctx, d, cl); err != nil {
			return err
		}
		if !oldEnabled && newEnabled {
			if err := enablePlugin(ctx, d, cl); err != nil {
				return err
			}
		}
		return nil
	}
	// update only "enabled"
	if d.HasChange("enabled") {
		if newEnabled {
			if err := enablePlugin(ctx, d, cl); err != nil {
				return err
			}
		} else {
			if err := disablePlugin(ctx, d, cl); err != nil {
				return err
			}
		}
	}
	return nil
}

func resourceDockerPluginUpdate(d *schema.ResourceData, meta interface{}) error {
	ctx := context.Background()
	if err := pluginUpdate(ctx, d, meta); err != nil {
		return err
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
