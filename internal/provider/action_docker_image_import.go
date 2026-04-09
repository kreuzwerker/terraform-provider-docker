package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DockerImageImportAction struct {
	providerConfig *ProviderConfig
}

type DockerImageImportActionModel struct {
	Source    types.String `tfsdk:"source"`
	Reference types.String `tfsdk:"reference"`
	Message   types.String `tfsdk:"message"`
	Changes   types.List   `tfsdk:"changes"`
	Platform  types.String `tfsdk:"platform"`
}

func (a *DockerImageImportAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_import"
}

func (a *DockerImageImportAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Import a tar archive or URL as a Docker image, similar to `docker image import`.",
		Attributes: map[string]actionschema.Attribute{
			"source": actionschema.StringAttribute{
				MarkdownDescription: "Path to a local tar archive or an http(s) URL containing the filesystem to import.",
				Required:            true,
			},
			"reference": actionschema.StringAttribute{
				MarkdownDescription: "Image name and optional tag to apply to the imported image, for example `my-image:latest`.",
				Required:            true,
			},
			"message": actionschema.StringAttribute{
				MarkdownDescription: "Optional message to store with the imported image.",
				Optional:            true,
			},
			"changes": actionschema.ListAttribute{
				MarkdownDescription: "Raw Dockerfile instructions to apply to the imported image.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"platform": actionschema.StringAttribute{
				MarkdownDescription: "Platform to assign to the imported image.",
				Optional:            true,
			},
		},
	}
}

func (a *DockerImageImportAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DockerImageImportAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker image import action invocation.")
		return
	}

	var config DockerImageImportActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Source.IsNull() || config.Source.IsUnknown() || strings.TrimSpace(config.Source.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid source", "Attribute `source` must be a non-empty path or URL.")
		return
	}

	if config.Reference.IsNull() || config.Reference.IsUnknown() || strings.TrimSpace(config.Reference.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid reference", "Attribute `reference` must be a non-empty image name.")
		return
	}

	sourceValue := strings.TrimSpace(config.Source.ValueString())
	referenceValue := strings.TrimSpace(config.Reference.ValueString())

	var changes []string
	if !config.Changes.IsNull() && !config.Changes.IsUnknown() {
		resp.Diagnostics.Append(config.Changes.ElementsAs(ctx, &changes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	sourceReader, err := openImageImportSource(ctx, sourceValue)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import source", err.Error())
		return
	}
	defer sourceReader.Close() // nolint:errcheck

	client, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	options := image.ImportOptions{
		Message:  config.Message.ValueString(),
		Changes:  changes,
		Platform: config.Platform.ValueString(),
	}

	responseBody, err := client.ImageImport(ctx, image.ImportSource{
		Source:     sourceReader,
		SourceName: "-",
	}, referenceValue, options)
	if err != nil {
		resp.Diagnostics.AddError("Docker image import failed", err.Error())
		return
	}
	defer responseBody.Close() // nolint:errcheck

	importOutput, err := io.ReadAll(responseBody)
	if err != nil {
		resp.Diagnostics.AddError("Docker image import output error", err.Error())
		return
	}

	if resp.SendProgress != nil {
		for _, line := range strings.Split(strings.TrimSpace(string(importOutput)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				resp.SendProgress(action.InvokeProgressEvent{Message: line})
			}
		}
	}

	inspectedImage, err := client.ImageInspect(ctx, referenceValue)
	if err != nil {
		resp.Diagnostics.AddError("Docker image inspect failed", err.Error())
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("imported_image_id=%s", inspectedImage.ID)})
	}
}

func openImageImportSource(ctx context.Context, source string) (io.ReadCloser, error) {
	parsedURL, err := url.Parse(source)
	if err == nil && parsedURL.Scheme != "" {
		switch parsedURL.Scheme {
		case "http", "https":
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
			if err != nil {
				return nil, err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close() // nolint:errcheck
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
				return nil, fmt.Errorf("unexpected HTTP status %s: %s", resp.Status, strings.TrimSpace(string(body)))
			}
			return resp.Body, nil
		case "file":
			path := parsedURL.Path
			if parsedURL.Host != "" {
				path = "//" + parsedURL.Host + path
			}
			return os.Open(path)
		}
	}

	return os.Open(source)
}
