package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &frameworkProvider{}
)

// frameworkProvider is the provider implementation using the Plugin Framework.
// This provider will be muxed with the SDK v2 provider to allow gradual migration.
type frameworkProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NewFrameworkProvider returns a new instance of the Plugin Framework provider.
func NewFrameworkProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &frameworkProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *frameworkProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "docker"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
// This schema must match the SDK v2 provider schema exactly for muxing to work.
func (p *frameworkProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The Docker daemon address",
				Optional:            true,
			},
			"context": schema.StringAttribute{
				MarkdownDescription: "The name of the Docker context to use. Can also be set via `DOCKER_CONTEXT` environment variable. Overrides the `host` if set.",
				Optional:            true,
			},
			"ssh_opts": schema.ListAttribute{
				MarkdownDescription: "Additional SSH option flags to be appended when using `ssh://` protocol",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ca_material": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded content of Docker host CA certificate",
				Optional:            true,
			},
			"cert_material": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded content of Docker client certificate",
				Optional:            true,
			},
			"key_material": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded content of Docker client private key",
				Optional:            true,
			},
			"cert_path": schema.StringAttribute{
				MarkdownDescription: "Path to directory with Docker TLS config",
				Optional:            true,
			},
			"disable_docker_daemon_check": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the provider will not check if the Docker daemon is running. This is useful for resources/data_sourcess that do not require a running Docker daemon, such as the data source `docker_registry_image`.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"registry_auth": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							MarkdownDescription: "Address of the registry",
							Required:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "Username for the registry. Defaults to `DOCKER_REGISTRY_USER` env variable if set.",
							Optional:            true,
						},
						"password": schema.StringAttribute{
							MarkdownDescription: "Password for the registry. Defaults to `DOCKER_REGISTRY_PASS` env variable if set.",
							Optional:            true,
							Sensitive:           true,
						},
						"config_file": schema.StringAttribute{
							MarkdownDescription: "Path to docker json file for registry auth. Defaults to `~/.docker/config.json`. If `DOCKER_CONFIG` env variable is set, the value of `DOCKER_CONFIG` is used as the path. `DOCKER_CONFIG` can be set to a directory (as per Docker CLI) or a file path directly. `config_file` has precedence over all other options.",
							Optional:            true,
						},
						"config_file_content": schema.StringAttribute{
							MarkdownDescription: "Plain content of the docker json file for registry auth. `config_file_content` has precedence over username/password.",
							Optional:            true,
						},
						"auth_disabled": schema.BoolAttribute{
							MarkdownDescription: "Setting this to `true` will tell the provider that this registry does not need authentication. Due to the docker internals, the provider will use dummy credentials (see https://github.com/kreuzwerker/terraform-provider-docker/issues/470 for more information). Defaults to `false`.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

// Configure prepares a Docker API client for data sources and resources.
// For now, configuration is handled by the SDK v2 provider through muxing.
func (p *frameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Configuration will be shared from the SDK v2 provider through the mux server
	// No configuration needed here yet
}

// Resources returns the provider's resource implementations.
// Initially empty - resources will be migrated from SDK v2 gradually.
func (p *frameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Resources will be added here as they are migrated from SDK v2
	}
}

// DataSources returns the provider's data source implementations.
// Initially empty - data sources will be migrated from SDK v2 gradually.
func (p *frameworkProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Data sources will be added here as they are migrated from SDK v2
	}
}
