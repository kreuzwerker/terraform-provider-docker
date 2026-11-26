package provider

import (
	"context"
	"fmt"
	"os/user"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const daemonKey = "daemon"

// daemonSchema is the per-resource connection block. It takes the same
// settings as the provider, without the provider's environment defaults:
// when the block is set, it is used on its own and nothing is inherited.
func daemonSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Docker daemon connection for this resource. When set, it is used instead of the provider configuration; nothing is inherited from the provider.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"host": {
					Type:         schema.TypeString,
					Optional:     true,
					ExactlyOneOf: []string{"daemon.0.host", "daemon.0.context"},
					Description:  "The Docker daemon address, e.g. `ssh://user@remote-host:22` or `tcp://remote-host:2376`.",
				},
				"context": {
					Type:         schema.TypeString,
					Optional:     true,
					ExactlyOneOf: []string{"daemon.0.host", "daemon.0.context"},
					Description:  "The name of the Docker context to use.",
				},
				"ssh_opts": {
					Type:        schema.TypeList,
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "Additional SSH option flags to be appended when using `ssh://` protocol.",
				},
				"ca_material": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "PEM-encoded content of Docker host CA certificate.",
				},
				"cert_material": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "PEM-encoded content of Docker client certificate.",
				},
				"key_material": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					Description: "PEM-encoded content of Docker client private key. Stored in state; prefer `cert_path`.",
				},
				"cert_path": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Path to directory with Docker TLS config.",
				},
				"disable_docker_daemon_check": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "If set to `true`, the provider will not check if the Docker daemon is running.",
				},
				"registry_auth": registryAuthSchema(false),
			},
		},
	}
}

// withDaemon adds the daemon block to a resource and replaces the resource
// when the daemon it lives on changes. Resources without an update get one
// that only reads: every other attribute is ForceNew, so the daemon block
// (e.g. new ssh_opts for the same host) is the only in-place change.
func withDaemon(r *schema.Resource) *schema.Resource {
	r.Schema[daemonKey] = daemonSchema()
	r.CustomizeDiff = daemonCustomizeDiff
	if r.UpdateContext == nil && r.Update == nil && r.UpdateWithoutTimeout == nil {
		r.UpdateContext = schema.UpdateContextFunc(r.ReadContext)
	}
	return r
}

// withDaemonDataSource adds the daemon block to a data source.
func withDaemonDataSource(r *schema.Resource) *schema.Resource {
	r.Schema[daemonKey] = daemonSchema()
	return r
}

// resolve returns the connection and registry credentials for a resource:
// its daemon block when set, the provider configuration otherwise.
func (c *ProviderConfig) resolve(d *schema.ResourceData) (Config, *AuthConfigs, error) {
	if d == nil {
		return *c.DefaultConfig, c.AuthConfigs, nil
	}

	raw, ok := d.GetOk(daemonKey)
	if !ok {
		return *c.DefaultConfig, c.AuthConfigs, nil
	}

	blocks := raw.([]interface{})
	if len(blocks) == 0 || blocks[0] == nil {
		return *c.DefaultConfig, c.AuthConfigs, nil
	}

	return expandDaemon(blocks[0].(map[string]interface{}))
}

func expandDaemon(m map[string]interface{}) (Config, *AuthConfigs, error) {
	host := m["host"].(string)

	if contextName := m["context"].(string); contextName != "" {
		usr, err := user.Current()
		if err != nil {
			return Config{}, nil, fmt.Errorf("could not determine current user to look for docker contexts: %w", err)
		}
		host, err = getContextHost(contextName, usr.HomeDir)
		if err != nil {
			return Config{}, nil, fmt.Errorf("error loading Docker context '%s': %w", contextName, err)
		}
	}

	rawSSHOpts := m["ssh_opts"].([]interface{})
	sshOpts := make([]string, len(rawSSHOpts))
	for i, v := range rawSSHOpts {
		sshOpts[i] = v.(string)
	}

	authConfigs := &AuthConfigs{}
	if set, ok := m["registry_auth"].(*schema.Set); ok && set.Len() > 0 {
		var err error
		authConfigs, err = providerSetToRegistryAuth(set)
		if err != nil {
			return Config{}, nil, fmt.Errorf("error loading daemon registry auth config: %w", err)
		}
	}

	return Config{
		Host:                     host,
		SSHOpts:                  sshOpts,
		Ca:                       m["ca_material"].(string),
		Cert:                     m["cert_material"].(string),
		Key:                      m["key_material"].(string),
		CertPath:                 m["cert_path"].(string),
		DisableDockerDaemonCheck: m["disable_docker_daemon_check"].(bool),
	}, authConfigs, nil
}

// daemonCustomizeDiff replaces a resource when the daemon it lives on changes.
// It never connects to a daemon. Resources whose prior state has no daemon
// block are adopted as they are, so adding the block to an existing or
// imported resource does not replace it.
func daemonCustomizeDiff(_ context.Context, d *schema.ResourceDiff, meta interface{}) error {
	if d.Id() == "" || !d.HasChange(daemonKey) {
		return nil
	}

	oldRaw, newRaw := d.GetChange(daemonKey)
	oldTarget, hadBlock := daemonTarget(oldRaw.([]interface{}))
	if !hadBlock {
		return nil
	}

	if !d.NewValueKnown(daemonKey+".0.host") || !d.NewValueKnown(daemonKey+".0.context") {
		return forceNewDaemon(d)
	}

	newTarget, hasBlock := daemonTarget(newRaw.([]interface{}))
	if !hasBlock {
		newTarget = meta.(*ProviderConfig).DefaultConfig.Host
	}

	if oldTarget != newTarget {
		return forceNewDaemon(d)
	}

	return nil
}

// forceNewDaemon marks the changed daemon key as ForceNew. ForceNew on the
// block itself only triggers when the number of blocks changes, so a changed
// host or context has to be marked on the nested key.
func forceNewDaemon(d *schema.ResourceDiff) error {
	for _, key := range []string{daemonKey + ".0.host", daemonKey + ".0.context", daemonKey} {
		if d.HasChange(key) {
			return d.ForceNew(key)
		}
	}
	return nil
}

// daemonTarget identifies the daemon a block points at, without resolving
// contexts: a context is compared by name.
func daemonTarget(blocks []interface{}) (string, bool) {
	if len(blocks) == 0 || blocks[0] == nil {
		return "", false
	}

	m := blocks[0].(map[string]interface{})
	if contextName, _ := m["context"].(string); contextName != "" {
		return "context:" + contextName, true
	}

	host, _ := m["host"].(string)
	return host, true
}
