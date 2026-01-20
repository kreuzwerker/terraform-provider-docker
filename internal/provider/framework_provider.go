package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
// For now, this is empty as all configuration is handled by the SDK v2 provider.
// Once we start migrating resources/data sources, we may need to add configuration here.
func (p *frameworkProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Docker provider is used to interact with Docker resources. " +
			"This is the Plugin Framework implementation that will gradually replace the SDK v2 implementation.",
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
