package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.Errorf("failed to create Docker client: %v", err)
	}

	pluginName := d.Get("name").(string)
	alias := d.Get("alias").(string)
	log.Printf("[DEBUG] Install a Docker plugin %s", pluginName)
	opts := types.PluginInstallOptions{
		RemoteRef:            pluginName,
		AcceptAllPermissions: d.Get("grant_all_permissions").(bool),
		Disabled:             !d.Get("enabled").(bool),
		// TODO suzuki-shunsuke: support other settings
		Args: getDockerPluginEnv(d.Get("env")),
	}
	if v, ok := d.GetOk("grant_permissions"); ok {
		opts.AcceptPermissionsFunc = getDockerPluginGrantPermissions(v)
	}
	body, err := client.PluginInstall(ctx, alias, opts)
	if err != nil {
		return diag.Errorf("install a Docker plugin "+pluginName+": %w", err)
	}
	_, _ = io.ReadAll(body)
	key := pluginName
	if alias != "" {
		key = alias
	}
	plugin, _, err := client.PluginInspectWithRaw(ctx, key)
	if err != nil {
		return diag.Errorf("inspect a Docker plugin "+key+": %w", err)
	}
	setDockerPlugin(d, plugin)
	return nil
}

func resourceDockerPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.Errorf("failed to create Docker client: %v", err)
	}

	pluginID := d.Id()
	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginID)
	if err != nil {
		log.Printf("[DEBUG] Inspect a Docker plugin "+pluginID+": %w", err)
		d.SetId("")
		return nil
	}

	jsonObj, _ := json.MarshalIndent(plugin, "", "\t")
	log.Printf("[DEBUG] Docker plugin inspect from readFunc: %s", jsonObj)

	setDockerPlugin(d, plugin)
	return nil
}

func resourceDockerPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.Errorf("failed to create Docker client: %v", err)
	}

	pluginID := d.Id()
	log.Printf("[DEBUG] Remove a Docker plugin %s", pluginID)
	if err := client.PluginRemove(ctx, pluginID, types.PluginRemoveOptions{
		Force: d.Get("force_destroy").(bool),
	}); err != nil {
		return diag.Errorf("remove the Docker plugin %s: %v", pluginID, err)
	}
	return nil
}

// Helpers
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

func normalizePluginName(name string) (string, error) {
	ref, err := reference.ParseAnyReference(name)
	if err != nil {
		return "", fmt.Errorf("parse the plugin name: %w", err)
	}
	return complementTag(ref.String()), nil
}

func diffSuppressFuncPluginName(k, oldV, newV string, d *schema.ResourceData) bool {
	o, err := normalizePluginName(oldV)
	if err != nil {
		return false
	}
	n, err := normalizePluginName(newV)
	if err != nil {
		return false
	}
	return o == n
}

func validateFuncPluginName(val interface{}, key string) (warns []string, errs []error) {
	if _, err := normalizePluginName(val.(string)); err != nil {
		return warns, append(errs, fmt.Errorf("%s is invalid: %w", key, err))
	}
	return
}

func getDockerPluginGrantPermissions(src interface{}) func(context.Context, types.PluginPrivileges) (bool, error) {
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
	return func(context context.Context, privileges types.PluginPrivileges) (bool, error) {
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

func setDockerPlugin(d *schema.ResourceData, plugin *types.Plugin) {
	d.SetId(plugin.ID)
	d.Set("plugin_reference", plugin.PluginReference)
	d.Set("alias", plugin.Name)
	d.Set("name", plugin.PluginReference)
	d.Set("enabled", plugin.Enabled)
	// TODO suzuki-shunsuke support other settings
	// https://docs.docker.com/engine/reference/commandline/plugin_set/#extended-description
	// source of mounts .Settings.Mounts
	// path of devices .Settings.Devices
	// args .Settings.Args
	d.Set("env", plugin.Settings.Env)
}

func disablePlugin(ctx context.Context, d *schema.ResourceData, cl *client.Client) error {
	pluginID := d.Id()
	log.Printf("[DEBUG] Disable a Docker plugin %s", pluginID)
	if err := cl.PluginDisable(ctx, pluginID, types.PluginDisableOptions{
		Force: d.Get("force_disable").(bool),
	}); err != nil {
		return fmt.Errorf("disable the Docker plugin %s: %w", pluginID, err)
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
	log.Printf("[DEBUG] Update settings of a Docker plugin %s", pluginID)
	// currently, only environment variables are supported.
	// TODO support other args
	// https://docs.docker.com/engine/reference/commandline/plugin_set/#extended-description
	// source of mounts .Settings.Mounts
	// path of devices .Settings.Devices
	// args .Settings.Args
	if err := cl.PluginSet(ctx, pluginID, getDockerPluginEnv(d.Get("env"))); err != nil {
		return fmt.Errorf("modify settings for the Docker plugin "+pluginID+": %w", err)
	}
	return nil
}

func pluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) (gErr error) {
	cl, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	o, n := d.GetChange("enabled")
	oldEnabled, newEnabled := o.(bool), n.(bool)
	if d.HasChange("env") {
		if oldEnabled {
			// To update the plugin settings, the plugin must be disabled
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

func resourceDockerPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := pluginUpdate(ctx, d, meta); err != nil {
		return diag.FromErr(err)
	}
	// call the read function to update the resource's state.
	// https://learn.hashicorp.com/tutorials/terraform/provider-update?in=terraform/providers#implement-update
	return resourceDockerPluginRead(ctx, d, meta)
}
