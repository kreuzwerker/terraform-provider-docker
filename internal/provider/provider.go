package provider

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:///var/run/docker.sock"),
					Description: "The Docker daemon address",
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
					Type:     schema.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Address of the registry",
							},

							"username": {
								Type:          schema.TypeString,
								Optional:      true,
								ConflictsWith: []string{"registry_auth.config_file", "registry_auth.config_file_content"},
								DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
								Description:   "Username for the registry",
							},

							"password": {
								Type:          schema.TypeString,
								Optional:      true,
								Sensitive:     true,
								ConflictsWith: []string{"registry_auth.config_file", "registry_auth.config_file_content"},
								DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
								Description:   "Password for the registry",
							},

							"config_file": {
								Type:          schema.TypeString,
								Optional:      true,
								ConflictsWith: []string{"registry_auth.username", "registry_auth.password", "registry_auth.config_file_content"},
								DefaultFunc:   schema.EnvDefaultFunc("DOCKER_CONFIG", "~/.docker/config.json"),
								Description:   "Path to docker json file for registry auth",
							},

							"config_file_content": {
								Type:          schema.TypeString,
								Optional:      true,
								ConflictsWith: []string{"registry_auth.username", "registry_auth.password", "registry_auth.config_file"},
								Description:   "Plain content of the docker json file for registry auth",
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
			},

			DataSourcesMap: map[string]*schema.Resource{
				"docker_registry_image": dataSourceDockerRegistryImage(),
				"docker_network":        dataSourceDockerNetwork(),
				"docker_plugin":         dataSourceDockerPlugin(),
				"docker_image":          dataSourceDockerImage(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := Config{
			Host:     d.Get("host").(string),
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

		if v, ok := d.GetOk("registry_auth"); ok { // TODO load them anyway
			authConfigs, err = providerSetToRegistryAuth(v.([]interface{}))

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
func providerSetToRegistryAuth(authList []interface{}) (*AuthConfigs, error) {
	authConfigs := AuthConfigs{
		Configs: make(map[string]types.AuthConfig),
	}

	for _, authInt := range authList {
		auth := authInt.(map[string]interface{})
		authConfig := types.AuthConfig{}
		authConfig.ServerAddress = normalizeRegistryAddress(auth["address"].(string))
		registryHostname := convertToHostname(authConfig.ServerAddress)

		// For each registry_auth block, generate an AuthConfiguration using either
		// username/password or the given config file
		if username, ok := auth["username"]; ok && username.(string) != "" {
			log.Println("[DEBUG] Using username for registry auths:", username)
			authConfig.Username = auth["username"].(string)
			authConfig.Password = auth["password"].(string)

			// Note: check for config_file_content first because config_file has a default which would be used
			// nevertheless config_file_content is set or not. The default has to be kept to check for the
			// environment variable and to be backwards compatible
		} else if configFileContent, ok := auth["config_file_content"]; ok && configFileContent.(string) != "" {
			log.Println("[DEBUG] Parsing file content for registry auths:", configFileContent.(string))
			r := strings.NewReader(configFileContent.(string))

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
		} else if configFile, ok := auth["config_file"]; ok && configFile.(string) != "" {
			filePath := configFile.(string)
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
				continue
			}
			c, err := loadConfigFile(r)
			if err != nil {
				continue
			}
			authFileConfig, err := c.GetAuthConfig(registryHostname)
			if err != nil {
				continue
			}
			authConfig.Username = authFileConfig.Username
			authConfig.Password = authFileConfig.Password
		}

		authConfigs.Configs[authConfig.ServerAddress] = authConfig
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
