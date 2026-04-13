package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dockerContainersDataSource{}
	_ datasource.DataSourceWithConfigure = &dockerContainersDataSource{}
)

type dockerContainersDataSource struct {
	providerConfig *ProviderConfig
}

type dockerContainersDataSourceModel struct {
	ID         types.String                     `tfsdk:"id"`
	Containers []dockerContainerDataSourceModel `tfsdk:"containers"`
}

type dockerContainerDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Names   types.List   `tfsdk:"names"`
	Image   types.String `tfsdk:"image"`
	ImageID types.String `tfsdk:"image_id"`
	Command types.String `tfsdk:"command"`
	Created types.Int64  `tfsdk:"created"`
	State   types.String `tfsdk:"state"`
	Status  types.String `tfsdk:"status"`
	Labels  types.Map    `tfsdk:"labels"`
}

func NewDockerContainersDataSource() datasource.DataSource {
	return &dockerContainersDataSource{}
}

func (d *dockerContainersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containers"
}

func (d *dockerContainersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Docker containers from the local Docker daemon.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source.",
				Computed:            true,
			},
			"containers": schema.ListNestedAttribute{
				MarkdownDescription: "List of Docker containers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The Docker container ID.",
							Computed:            true,
						},
						"names": schema.ListAttribute{
							MarkdownDescription: "The container names.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"image": schema.StringAttribute{
							MarkdownDescription: "The image used by the container.",
							Computed:            true,
						},
						"image_id": schema.StringAttribute{
							MarkdownDescription: "The image ID used by the container.",
							Computed:            true,
						},
						"command": schema.StringAttribute{
							MarkdownDescription: "The command used to run the container.",
							Computed:            true,
						},
						"created": schema.Int64Attribute{
							MarkdownDescription: "The Unix timestamp when the container was created.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "The container state.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The container status.",
							Computed:            true,
						},
						"labels": schema.MapAttribute{
							MarkdownDescription: "Labels applied to the container.",
							ElementType:         types.StringType,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *dockerContainersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dockerContainersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker_containers data source.")
		return
	}

	client, err := d.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	apiContainers, err := client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		resp.Diagnostics.AddError("Docker container list failed", err.Error())
		return
	}

	sort.Slice(apiContainers, func(i, j int) bool {
		return apiContainers[i].ID < apiContainers[j].ID
	})

	containers, diags := flattenDockerContainers(ctx, apiContainers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := dockerContainersDataSourceModel{
		ID:         types.StringValue("docker_containers"),
		Containers: containers,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func flattenDockerContainers(ctx context.Context, apiContainers []container.Summary) ([]dockerContainerDataSourceModel, diag.Diagnostics) {
	result := make([]dockerContainerDataSourceModel, 0, len(apiContainers))
	diags := make(diag.Diagnostics, 0)

	for _, apiContainer := range apiContainers {
		names, namesDiags := types.ListValueFrom(ctx, types.StringType, apiContainer.Names)
		diags.Append(namesDiags...)
		if namesDiags.HasError() {
			return nil, diags
		}

		labels, labelsDiags := types.MapValueFrom(ctx, types.StringType, apiContainer.Labels)
		diags.Append(labelsDiags...)
		if labelsDiags.HasError() {
			return nil, diags
		}

		result = append(result, dockerContainerDataSourceModel{
			ID:      types.StringValue(apiContainer.ID),
			Names:   names,
			Image:   types.StringValue(apiContainer.Image),
			ImageID: types.StringValue(apiContainer.ImageID),
			Command: types.StringValue(apiContainer.Command),
			Created: types.Int64Value(apiContainer.Created),
			State:   types.StringValue(apiContainer.State),
			Status:  types.StringValue(apiContainer.Status),
			Labels:  labels,
		})
	}

	return result, diags
}
