package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DockerContainerExportAction struct {
	providerConfig *ProviderConfig
}

type DockerContainerExportActionModel struct {
	Container types.String `tfsdk:"container"`
	Output    types.String `tfsdk:"output"`
}

func (a *DockerContainerExportAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_export"
}

func (a *DockerContainerExportAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Export a container filesystem as a tar archive, similar to `docker container export`.",
		Attributes: map[string]actionschema.Attribute{
			"container": actionschema.StringAttribute{
				MarkdownDescription: "Container name or ID to export.",
				Required:            true,
			},
			"output": actionschema.StringAttribute{
				MarkdownDescription: "Path to the output tar archive file.",
				Required:            true,
			},
		},
	}
}

func (a *DockerContainerExportAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DockerContainerExportAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker container export action invocation.")
		return
	}

	var config DockerContainerExportActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerName := strings.TrimSpace(config.Container.ValueString())
	if config.Container.IsNull() || config.Container.IsUnknown() || containerName == "" {
		resp.Diagnostics.AddError("Invalid container", "Attribute `container` must be a non-empty container name or ID.")
		return
	}

	outputPath := strings.TrimSpace(config.Output.ValueString())
	if config.Output.IsNull() || config.Output.IsUnknown() || outputPath == "" {
		resp.Diagnostics.AddError("Invalid output", "Attribute `output` must be a non-empty file path.")
		return
	}

	dockerClient, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	exportStream, err := dockerClient.ContainerExport(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError("Docker container export failed", err.Error())
		return
	}
	defer exportStream.Close() // nolint:errcheck

	outputFile, err := os.Create(outputPath)
	if err != nil {
		resp.Diagnostics.AddError("Output file error", err.Error())
		return
	}
	defer outputFile.Close() // nolint:errcheck

	writtenBytes, err := io.Copy(outputFile, exportStream)
	if err != nil {
		resp.Diagnostics.AddError("Docker container export output error", err.Error())
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("exported_container=%s output=%s bytes_written=%d", containerName, outputPath, writtenBytes)})
	}
}
