package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DockerSystemPruneAction struct {
	providerConfig *ProviderConfig
}

type DockerSystemPruneActionModel struct {
	All     types.Bool `tfsdk:"all"`
	Volumes types.Bool `tfsdk:"volumes"`
	Filter  types.List `tfsdk:"filter"`
}

func (a *DockerSystemPruneAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_prune"
}

func (a *DockerSystemPruneAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Remove unused Docker objects, similar to `docker system prune` (without interactive confirmation).",
		Attributes: map[string]actionschema.Attribute{
			"all": actionschema.BoolAttribute{
				MarkdownDescription: "Remove all unused images, not just dangling ones.",
				Optional:            true,
			},
			"filter": actionschema.ListAttribute{
				MarkdownDescription: "Provide filter values in `key=value` format. Can be specified multiple times.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"volumes": actionschema.BoolAttribute{
				MarkdownDescription: "Prune anonymous volumes in addition to containers, networks, images, and build cache.",
				Optional:            true,
			},
		},
	}
}

func (a *DockerSystemPruneAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DockerSystemPruneAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker system prune action invocation.")
		return
	}

	var config DockerSystemPruneActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var filterExpressions []string
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		resp.Diagnostics.Append(config.Filter.ElementsAs(ctx, &filterExpressions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	pruneFilters, err := parseSystemPruneFilterExpressions(filterExpressions)
	if err != nil {
		resp.Diagnostics.AddError("Invalid filter", err.Error())
		return
	}

	imageFilters := pruneFilters.Clone()
	if config.All.ValueBool() {
		imageFilters.Add("dangling", "false")
	}

	client, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	containerReport, err := client.ContainersPrune(ctx, pruneFilters)
	if err != nil {
		resp.Diagnostics.AddError("Docker container prune failed", err.Error())
		return
	}

	imageReport, err := client.ImagesPrune(ctx, imageFilters)
	if err != nil {
		resp.Diagnostics.AddError("Docker image prune failed", err.Error())
		return
	}

	networkReport, err := client.NetworksPrune(ctx, pruneFilters)
	if err != nil {
		resp.Diagnostics.AddError("Docker network prune failed", err.Error())
		return
	}

	buildCacheReport, err := client.BuildCachePrune(ctx, build.CachePruneOptions{
		Filters: pruneFilters,
	})
	if err != nil {
		resp.Diagnostics.AddError("Docker build cache prune failed", err.Error())
		return
	}

	var volumeObjectsDeleted int
	totalReclaimedBytes := containerReport.SpaceReclaimed + imageReport.SpaceReclaimed + buildCacheReport.SpaceReclaimed
	if config.Volumes.ValueBool() {
		volumeReport, volumeErr := client.VolumesPrune(ctx, pruneFilters)
		if volumeErr != nil {
			resp.Diagnostics.AddError("Docker volume prune failed", volumeErr.Error())
			return
		}
		volumeObjectsDeleted = len(volumeReport.VolumesDeleted)
		totalReclaimedBytes += volumeReport.SpaceReclaimed
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf(
				"pruned_containers=%d pruned_images=%d pruned_networks=%d pruned_build_cache_entries=%d pruned_volumes=%d space_reclaimed_bytes=%d",
				len(containerReport.ContainersDeleted),
				len(imageReport.ImagesDeleted),
				len(networkReport.NetworksDeleted),
				len(buildCacheReport.CachesDeleted),
				volumeObjectsDeleted,
				totalReclaimedBytes,
			),
		})
	}
}

func parseSystemPruneFilterExpressions(items []string) (filters.Args, error) {
	filterArgs := filters.NewArgs()

	for _, item := range items {
		filterExpression := strings.TrimSpace(item)
		if filterExpression == "" {
			return filters.Args{}, fmt.Errorf("empty filter expression")
		}

		key, value, ok := strings.Cut(filterExpression, "=")
		if !ok {
			return filters.Args{}, fmt.Errorf("filter %q must be in key=value format", filterExpression)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return filters.Args{}, fmt.Errorf("filter %q has an empty key", filterExpression)
		}

		filterArgs.Add(key, value)
	}

	return filterArgs, nil
}
