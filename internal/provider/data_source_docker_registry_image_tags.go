package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dockerRegistryImageTagsDataSource{}
	_ datasource.DataSourceWithConfigure = &dockerRegistryImageTagsDataSource{}
)

type dockerRegistryImageTagsDataSource struct {
	providerConfig *ProviderConfig
}

type dockerRegistryImageTagsDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	StrictSemver       types.Bool   `tfsdk:"strict_semver"`
	Tags               types.List   `tfsdk:"tags"`
}

func NewDockerRegistryImageTagsDataSource() datasource.DataSource {
	return &dockerRegistryImageTagsDataSource{}
}

func (d *dockerRegistryImageTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_image_tags"
}

func (d *dockerRegistryImageTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the tags available for an image in a Docker registry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source.",
				Computed:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Docker image repository, including any tag or digest. For example, `alpine:latest`.",
				Required:            true,
			},

			"insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "If `true`, the verification of TLS certificates of the server/registry is disabled. Defaults to `false`.",
				Optional:            true,
			},

			"strict_semver": schema.BoolAttribute{
				MarkdownDescription: "If `true`, only stable semantic version tags are returned. Prerelease tags such as `1.2.3-rc.1` are excluded as well as any other tags that do not conform to the semantic versioning specification. Defaults to `false`.",
				Optional:            true,
			},

			"tags": schema.ListAttribute{
				MarkdownDescription: "List of available Docker image tags matching the specified criteria.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *dockerRegistryImageTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.providerConfig = providerConfig
}

func (d *dockerRegistryImageTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker_registry_image_tags data source.")
		return
	}

	var config dockerRegistryImageTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pullOpts := parseImageOptions(config.Name.ValueString())

	authConfig, err := getAuthConfigForRegistry(pullOpts.Registry, d.providerConfig)
	if err != nil {
		// The user did not provide a credential for this registry.
		// But there are many registries where you can pull without a credential.
		// We are setting default values for the authConfig here.
		authConfig.Username = ""
		authConfig.Password = ""
		authConfig.ServerAddress = "https://" + pullOpts.Registry
	}

	tags, err := getImageTags(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, authConfig.Username, authConfig.Password, config.InsecureSkipVerify.ValueBool(), false)
	if err != nil {
		tags, err = getImageTags(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, authConfig.Username, authConfig.Password, config.InsecureSkipVerify.ValueBool(), true)
		if err != nil {
			resp.Diagnostics.AddError("Docker registry tags lookup failed", fmt.Sprintf("Got error when attempting to fetch image tags for %s from registry: %s", config.Name.ValueString(), err))
			return
		}
	}

	if config.StrictSemver.ValueBool() {
		tags = filterStrictSemverTags(tags)
	}

	sort.Strings(tags)
	tagsList, diags := types.ListValueFrom(ctx, types.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := dockerRegistryImageTagsDataSourceModel{
		ID:                 types.StringValue(fmt.Sprintf("%s/%s", pullOpts.Registry, pullOpts.Repository)),
		Name:               config.Name,
		InsecureSkipVerify: config.InsecureSkipVerify,
		StrictSemver:       config.StrictSemver,
		Tags:               tagsList,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getImageTags(registry, registryWithProtocol, image, username, password string, insecureSkipVerify, fallback bool) ([]string, error) {
	client := buildHttpClientForRegistry(registryWithProtocol, insecureSkipVerify)

	req, err := setupHTTPRequestForTagCollection(registry, registryWithProtocol, image, "", username, password, fallback)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error during registry request: %s", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		return getTagsFromResponse(resp)

	// Either OAuth is required or the basic auth creds were invalid
	case http.StatusUnauthorized:
		auth, err := parseAuthHeader(resp.Header.Get("www-authenticate"))
		if err != nil {
			return nil, fmt.Errorf("bad credentials: %s", resp.Status)
		}

		token, err := getAuthToken(auth, username, password, "repository:"+image+":pull", client)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return doTagsRequest(req, client)

	// Some unexpected status was given, return an error
	default:
		return nil, fmt.Errorf("got bad response from registry: %s", resp.Status)
	}
}

func doTagsRequest(req *http.Request, client *http.Client) ([]string, error) {
	tagsResponse, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during registry request: %s", err)
	}
	defer tagsResponse.Body.Close() // nolint:errcheck

	if tagsResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got bad response from registry: %s", tagsResponse.Status)
	}

	return getTagsFromResponse(tagsResponse)
}

func getTagsFromResponse(response *http.Response) ([]string, error) {
	if response.Body == nil {
		return nil, fmt.Errorf("error reading registry response body: response body is nil")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading registry response body: %s", err)
	}

	tagsResponse := &TagsResponse{}
	err = json.Unmarshal(body, tagsResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing tags response: %s", err)
	}

	return tagsResponse.Tags, nil
}

type TagsResponse struct {
	Tags []string `json:"tags"`
}

func filterStrictSemverTags(tags []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		version, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}

		if version.Prerelease() != "" {
			continue
		}

		filtered = append(filtered, tag)
	}

	return filtered
}
