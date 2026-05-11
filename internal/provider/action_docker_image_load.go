package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DockerImageLoadAction struct {
	providerConfig *ProviderConfig
}

type DockerImageLoadActionModel struct {
	Source   types.String `tfsdk:"source"`
	Quiet    types.Bool   `tfsdk:"quiet"`
	Platform types.String `tfsdk:"platform"`
}

func (a *DockerImageLoadAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_load"
}

func (a *DockerImageLoadAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Load a Docker image from a tar archive file, similar to `docker image load`.",
		Attributes: map[string]actionschema.Attribute{
			"source": actionschema.StringAttribute{
				MarkdownDescription: "Path to a local image tar archive file.",
				Required:            true,
			},
			"quiet": actionschema.BoolAttribute{
				MarkdownDescription: "Suppress progress details in Docker daemon output.",
				Optional:            true,
			},
			"platform": actionschema.StringAttribute{
				MarkdownDescription: "Optional platform to load from a multi-platform image, for example `linux/amd64`.",
				Optional:            true,
			},
		},
	}
}

func (a *DockerImageLoadAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.providerConfig = providerConfig
}

func (a *DockerImageLoadAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker image load action invocation.")
		return
	}

	var config DockerImageLoadActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourcePath := strings.TrimSpace(config.Source.ValueString())
	if config.Source.IsNull() || config.Source.IsUnknown() || sourcePath == "" {
		resp.Diagnostics.AddError("Invalid source", "Attribute `source` must be a non-empty file path.")
		return
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		resp.Diagnostics.AddError("Invalid source", err.Error())
		return
	}
	defer sourceFile.Close() // nolint:errcheck

	var loadOptions []client.ImageLoadOption
	if !config.Quiet.IsNull() && !config.Quiet.IsUnknown() {
		loadOptions = append(loadOptions, client.ImageLoadWithQuiet(config.Quiet.ValueBool()))
	}

	if !config.Platform.IsNull() && !config.Platform.IsUnknown() {
		parsedPlatform, parseErr := parseOptionalPlatform(config.Platform.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid platform", parseErr.Error())
			return
		}
		if parsedPlatform != nil {
			loadOptions = append(loadOptions, client.ImageLoadWithPlatforms(*parsedPlatform))
		}
	}

	dockerClient, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	loadResponse, err := dockerClient.ImageLoad(ctx, sourceFile, loadOptions...)
	if err != nil {
		resp.Diagnostics.AddError("Docker image load failed", err.Error())
		return
	}
	defer loadResponse.Body.Close() // nolint:errcheck

	loadOutput, err := io.ReadAll(loadResponse.Body)
	if err != nil {
		resp.Diagnostics.AddError("Docker image load output error", err.Error())
		return
	}

	if resp.SendProgress != nil {
		for _, line := range strings.Split(strings.TrimSpace(string(loadOutput)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				resp.SendProgress(action.InvokeProgressEvent{Message: line})
			}
		}
	}
}
