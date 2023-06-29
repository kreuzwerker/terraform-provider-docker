package provider

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

// New creates the Docker provider
func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"host": {
					Type:     schema.TypeString,
					Required: true,
					DefaultFunc: func() (interface{}, error) {
						if v := os.Getenv("DOCKER_HOST"); v != "" {
							return v, nil
						}
						if runtime.GOOS == "windows" {
							return "npipe:////./pipe/docker_engine", nil
						}
						return "unix:///var/run/docker.sock", nil
					},
					Description: "The Docker daemon address",
				},
				"ssh_opts": {
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
					DefaultFunc: func() (interface{}, error) {
						if v := os.Getenv("DOCKER_SSH_OPTS"); v != "" {
							return strings.Fields(v), nil
						}

						return nil, nil
					},
					Description: "Additional SSH option flags to be appended when using `ssh://` protocol",
				},
				"ca_material": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("DOCKER_CA_MATERIAL", ""),
					Description: "PEM-encoded content of Docker host CA certificate",
				},
				"cert_material": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_MATERIAL", ""),
					Description: "PEM-encoded content of Docker client certificate",
				},
				"key_material": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("DOCKER_KEY_MATERIAL", ""),
					Description: "PEM-encoded content of Docker client private key",
				},

				"cert_path": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_PATH", ""),
					Description: "Path to directory with Docker TLS config",
				},

				"registry_auth": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Type:         schema.TypeString,
								Required:     true,
								ValidateFunc: validation.StringIsNotEmpty,
								Description:  "Address of the registry",
							},

							"username": {
								Type:        schema.TypeString,
								Optional:    true,
								DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
								Description: "Username for the registry. Defaults to `DOCKER_REGISTRY_USER` env variable if set.",
							},

							"password": {
								Type:        schema.TypeString,
								Optional:    true,
								Sensitive:   true,
								DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
								Description: "Password for the registry. Defaults to `DOCKER_REGISTRY_PASS` env variable if set.",
							},

							"config_file": {
								Type:        schema.TypeString,
								Optional:    true,
								DefaultFunc: schema.EnvDefaultFunc("DOCKER_CONFIG", "~/.docker/config.json"),
								Description: "Path to docker json file for registry auth. Defaults to `~/.docker/config.json`. If `DOCKER_CONFIG` is set, the value of `DOCKER_CONFIG` is used as the path. `config_file` has predencen over all other options.",
							},

							"config_file_content": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Plain content of the docker json file for registry auth. `config_file_content` has precedence over username/password.",
							},
							"auth_disabled": {
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
								Description: "Setting this to `true` will tell the provider that this registry does not need authentication. Due to the docker internals, the provider will use dummy credentials (see https://github.com/kreuzwerker/terraform-provider-docker/issues/470 for more information). Defaults to `false`.",
							},
						},
					},
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"docker_container":      resourceDockerContainer(),
				"docker_image":          resourceDockerImage(),
				"docker_registry_image": resourceDockerRegistryImage(),
				"docker_network":        resourceDockerNetwork(),
				"docker_volume":         resourceDockerVolume(),
				"docker_config":         resourceDockerConfig(),
				"docker_secret":         resourceDockerSecret(),
				"docker_service":        resourceDockerService(),
				"docker_plugin":         resourceDockerPlugin(),
				"docker_tag":            resourceDockerTag(),
			},

			DataSourcesMap: map[string]*schema.Resource{
				"docker_registry_image": dataSourceDockerRegistryImage(),
				"docker_network":        dataSourceDockerNetwork(),
				"docker_plugin":         dataSourceDockerPlugin(),
				"docker_image":          dataSourceDockerImage(),
				"docker_logs":           dataSourceDockerLogs(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		SSHOptsI := d.Get("ssh_opts").([]interface{})
		SSHOpts := make([]string, len(SSHOptsI))
		for i, s := range SSHOptsI {
			SSHOpts[i] = s.(string)
		}
		config := Config{
			Host:     d.Get("host").(string),
			SSHOpts:  SSHOpts,
			Ca:       d.Get("ca_material").(string),
			Cert:     d.Get("cert_material").(string),
			Key:      d.Get("key_material").(string),
			CertPath: d.Get("cert_path").(string),
		}

		client, err := config.NewClient()
		if err != nil {
			return nil, diag.Errorf("Error initializing Docker client: %s", err)
		}

		_, err = client.Ping(ctx)
		if err != nil {
			return nil, diag.Errorf("Error pinging Docker server: %s", err)
		}

		authConfigs := &AuthConfigs{}

		if v, ok := d.GetOk("registry_auth"); ok {
			authConfigs, err = providerSetToRegistryAuth(v.(*schema.Set))
			if err != nil {
				return nil, diag.Errorf("Error loading registry auth config: %s", err)
			}
		}

		providerConfig := ProviderConfig{
			DockerClient: client,
			AuthConfigs:  authConfigs,
		}

		return &providerConfig, nil
	}
}

// AuthConfigs represents authentication options to use for the
// PushImage method accommodating the new X-Registry-Config header
type AuthConfigs struct {
	Configs map[string]types.AuthConfig `json:"configs"`
}

// Take the given registry_auth schemas and return a map of registry auth configurations
func providerSetToRegistryAuth(authList *schema.Set) (*AuthConfigs, error) {
	authConfigs := AuthConfigs{
		Configs: make(map[string]types.AuthConfig),
	}

	for _, auth := range authList.List() {
		authConfig := types.AuthConfig{}
		address := auth.(map[string]interface{})["address"].(string)
		authConfig.ServerAddress = normalizeRegistryAddress(address)
		registryHostname := convertToHostname(authConfig.ServerAddress)

		username, ok := auth.(map[string]interface{})["username"].(string)
		password := auth.(map[string]interface{})["password"].(string)

		// If auth is disabled, set the auth config to any user/password combination
		// See https://github.com/kreuzwerker/terraform-provider-docker/issues/470 for more information
		if auth.(map[string]interface{})["auth_disabled"].(bool) {
			log.Printf("[DEBUG] Auth disabled for registry %s", registryHostname)
			username = "username"
			password = "password"
		}

		// For each registry_auth block, generate an AuthConfiguration using either
		// username/password or the given config file
		if ok && username != "" {
			log.Println("[DEBUG] Using username for registry auths:", username)

			if isECRRepositoryURL(registryHostname) {
				password = normalizeECRPasswordForDockerCLIUsage(password)
			}
			authConfig.Username = username
			authConfig.Password = password

			// Note: check for config_file_content first because config_file has a default which would be used
			// nevertheless config_file_content is set or not. The default has to be kept to check for the
			// environment variable and to be backwards compatible
		} else if configFileContent, ok := auth.(map[string]interface{})["config_file_content"].(string); ok && configFileContent != "" {
			log.Println("[DEBUG] Parsing file content for registry auths:", configFileContent)
			r := strings.NewReader(configFileContent)

			c, err := loadConfigFile(r)
			if err != nil {
				return nil, fmt.Errorf("Error parsing docker registry config json: %v", err)
			}
			authFileConfig, err := c.GetAuthConfig(registryHostname)
			if err != nil {
				return nil, fmt.Errorf("couldn't find registry config for '%s' in file content", registryHostname)
			}
			authConfig.Username = authFileConfig.Username
			authConfig.Password = authFileConfig.Password

			// As last step we check if a config file path is given
		} else if configFile, ok := auth.(map[string]interface{})["config_file"].(string); ok && configFile != "" {
			filePath := configFile
			log.Println("[DEBUG] Parsing file for registry auths:", filePath)

			// We manually expand the path and do not use the 'pathexpand' interpolation function
			// because in the default of this varable we refer to '~/.docker/config.json'
			if strings.HasPrefix(filePath, "~/") {
				usr, err := user.Current()
				if err != nil {
					return nil, err
				}
				filePath = strings.Replace(filePath, "~", usr.HomeDir, 1)
			}
			r, err := os.Open(filePath)
			if err != nil {
				return nil, fmt.Errorf("could not open config file from filePath: %s. Error: %v", filePath, err)
			}
			c, err := loadConfigFile(r)
			if err != nil {
				return nil, fmt.Errorf("could not read and load config file: %v", err)
			}
			authFileConfig, err := c.GetAuthConfig(registryHostname)
			if err != nil {
				return nil, fmt.Errorf("could not get auth config (the credentialhelper did not work or was not found): %v", err)
			}
			authConfig.Username = authFileConfig.Username
			authConfig.Password = authFileConfig.Password
		}

		authConfigs.Configs[registryHostname] = authConfig
	}

	return &authConfigs, nil
}

func loadConfigFile(configData io.Reader) (*configfile.ConfigFile, error) {
	configFile := configfile.New("")
	if err := configFile.LoadFromReader(configData); err != nil {
		log.Println("[DEBUG] Error parsing registry config: ", err)
		log.Println("[DEBUG] Will try parsing from legacy format")
		if err := configFile.LegacyLoadFromReader(configData); err != nil {
			return nil, err
		}
	}
	return configFile, nil
}

// ConvertToHostname converts a registry url which has http|https prepended
// to just an hostname.
// Copied from github.com/docker/docker/registry.ConvertToHostname to reduce dependencies.
func convertToHostname(url string) string {
	stripped := url
	// DevSkim: ignore DS137138
	if strings.HasPrefix(url, "http://") {
		// DevSkim: ignore DS137138
		stripped = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		stripped = strings.TrimPrefix(url, "https://")
	}

	nameParts := strings.SplitN(stripped, "/", 2)

	return nameParts[0]
}
