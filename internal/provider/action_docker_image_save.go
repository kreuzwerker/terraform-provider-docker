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

type DockerImageSaveAction struct {
	providerConfig *ProviderConfig
}

type DockerImageSaveActionModel struct {
	Images   types.List   `tfsdk:"images"`
	Output   types.String `tfsdk:"output"`
	Platform types.String `tfsdk:"platform"`
}

func (a *DockerImageSaveAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_save"
}

func (a *DockerImageSaveAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Save one or more Docker images to a tar archive, similar to `docker image save`.",
		Attributes: map[string]actionschema.Attribute{
			"images": actionschema.ListAttribute{
				MarkdownDescription: "List of image names or IDs to include in the output archive.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"output": actionschema.StringAttribute{
				MarkdownDescription: "Path to the output tar archive file.",
				Required:            true,
			},
			"platform": actionschema.StringAttribute{
				MarkdownDescription: "Optional platform to save from a multi-platform image, for example `linux/amd64`.",
				Optional:            true,
			},
		},
	}
}

func (a *DockerImageSaveAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DockerImageSaveAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker image save action invocation.")
		return
	}

	var config DockerImageSaveActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var images []string
	resp.Diagnostics.Append(config.Images.ElementsAs(ctx, &images, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	imageReferences := make([]string, 0, len(images))
	for _, imageReference := range images {
		trimmed := strings.TrimSpace(imageReference)
		if trimmed == "" {
			resp.Diagnostics.AddError("Invalid images", "Attribute `images` must not contain empty image names.")
			return
		}

		imageReferences = append(imageReferences, trimmed)
	}

	if len(imageReferences) == 0 {
		resp.Diagnostics.AddError("Invalid images", "Attribute `images` must contain at least one image name or ID.")
		return
	}

	outputPath := strings.TrimSpace(config.Output.ValueString())
	if config.Output.IsNull() || config.Output.IsUnknown() || outputPath == "" {
		resp.Diagnostics.AddError("Invalid output", "Attribute `output` must be a non-empty file path.")
		return
	}

	var saveOptions []client.ImageSaveOption
	if !config.Platform.IsNull() && !config.Platform.IsUnknown() {
		parsedPlatform, err := parseOptionalPlatform(config.Platform.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid platform", err.Error())
			return
		}
		if parsedPlatform != nil {
			saveOptions = append(saveOptions, client.ImageSaveWithPlatforms(*parsedPlatform))
		}
	}

	dockerClient, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	imageTarStream, err := dockerClient.ImageSave(ctx, imageReferences, saveOptions...)
	if err != nil {
		resp.Diagnostics.AddError("Docker image save failed", err.Error())
		return
	}
	defer imageTarStream.Close() // nolint:errcheck

	outputFile, err := os.Create(outputPath)
	if err != nil {
		resp.Diagnostics.AddError("Output file error", err.Error())
		return
	}
	defer outputFile.Close() // nolint:errcheck

	writtenBytes, err := io.Copy(outputFile, imageTarStream)
	if err != nil {
		resp.Diagnostics.AddError("Docker image save output error", err.Error())
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("saved_images=%d output=%s bytes_written=%d", len(imageReferences), outputPath, writtenBytes)})
	}
}
