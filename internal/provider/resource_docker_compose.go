package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	composeapi "github.com/docker/compose/v2/pkg/api"
	compose "github.com/docker/compose/v2/pkg/compose"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &dockerComposeResource{}
	_ resource.ResourceWithConfigure = &dockerComposeResource{}
)

type dockerComposeResource struct {
	providerConfig *ProviderConfig
}

type dockerComposeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ConfigPaths      types.List   `tfsdk:"config_paths"`
	ProjectDirectory types.String `tfsdk:"project_directory"`
	ProjectName      types.String `tfsdk:"project_name"`
	Profiles         types.List   `tfsdk:"profiles"`
	EnvFiles         types.List   `tfsdk:"env_files"`
	RemoveOrphans    types.Bool   `tfsdk:"remove_orphans"`
	Wait             types.Bool   `tfsdk:"wait"`
	WaitTimeout      types.String `tfsdk:"wait_timeout"`
}

func NewDockerComposeResource() resource.Resource {
	return &dockerComposeResource{}
}

func (r *dockerComposeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compose"
}

func (r *dockerComposeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Docker Compose application using the Docker Compose Go packages. The resource loads one or more Compose files and applies them with the equivalent of `docker compose up`, then removes them with the equivalent of `docker compose down`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Compose project name used as the Terraform resource ID.",
				Computed:            true,
			},
			"config_paths": schema.ListAttribute{
				MarkdownDescription: "One or more Compose file paths, equivalent to repeating the `-f` flag with `docker compose`.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"project_directory": schema.StringAttribute{
				MarkdownDescription: "Optional project directory used as the Compose working directory. If omitted, Compose uses the directory of the first file in `config_paths`.",
				Optional:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "Optional Compose project name. If omitted, Compose derives the project name the same way as the CLI.",
				Optional:            true,
				Computed:            true,
			},
			"profiles": schema.ListAttribute{
				MarkdownDescription: "Optional list of Compose profiles to enable.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"env_files": schema.ListAttribute{
				MarkdownDescription: "Optional list of env files to load before parsing the Compose configuration. If omitted, Compose uses the default `.env` behavior.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"remove_orphans": schema.BoolAttribute{
				MarkdownDescription: "If `true`, remove containers for services that are no longer present in the Compose configuration during apply and destroy.",
				Optional:            true,
			},
			"wait": schema.BoolAttribute{
				MarkdownDescription: "If `true`, wait until services reach the running or healthy state before returning from apply.",
				Optional:            true,
			},
			"wait_timeout": schema.StringAttribute{
				MarkdownDescription: "Optional duration for `wait`, for example `30s` or `2m`.",
				Optional:            true,
			},
		},
	}
}

func (r *dockerComposeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.providerConfig = providerConfig
}

func (r *dockerComposeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerComposeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, service := r.prepareComposeProject(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	upOptions, ok := buildComposeUpOptions(plan, project, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	if err := service.Create(ctx, project, upOptions.Create); err != nil {
		resp.Diagnostics.AddError("Docker Compose create failed", err.Error())
		return
	}

	if err := service.Start(ctx, project.Name, upOptions.Start); err != nil {
		resp.Diagnostics.AddError("Docker Compose start failed", err.Error())
		return
	}

	plan.ProjectName = types.StringValue(project.Name)
	plan.ID = types.StringValue(project.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerComposeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerComposeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := strings.TrimSpace(state.ProjectName.ValueString())
	if projectName == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	service := r.newComposeService(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	containers, err := service.Ps(ctx, projectName, composeapi.PsOptions{All: true})
	if err != nil {
		if composeapi.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Docker Compose read failed", err.Error())
		return
	}

	if len(containers) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerComposeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerComposeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, service := r.prepareComposeProject(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	upOptions, ok := buildComposeUpOptions(plan, project, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	if err := service.Create(ctx, project, upOptions.Create); err != nil {
		resp.Diagnostics.AddError("Docker Compose update create failed", err.Error())
		return
	}

	if err := service.Start(ctx, project.Name, upOptions.Start); err != nil {
		resp.Diagnostics.AddError("Docker Compose update start failed", err.Error())
		return
	}

	plan.ProjectName = types.StringValue(project.Name)
	plan.ID = types.StringValue(project.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerComposeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerComposeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := strings.TrimSpace(state.ProjectName.ValueString())
	if projectName == "" {
		return
	}

	service := r.newComposeService(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var project *composetypes.Project
	loadedProject, diags := loadComposeProject(ctx, state)
	resp.Diagnostics.Append(diags...)
	if !diags.HasError() {
		project = loadedProject
	}

	if err := service.Down(ctx, projectName, composeapi.DownOptions{
		Project:       project,
		RemoveOrphans: state.RemoveOrphans.ValueBool(),
	}); err != nil && !composeapi.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Docker Compose destroy failed", err.Error())
	}
}

func (r *dockerComposeResource) prepareComposeProject(ctx context.Context, model dockerComposeResourceModel, diags *diag.Diagnostics) (*composetypes.Project, composeapi.Service) {
	project, loadDiags := loadComposeProject(ctx, model)
	diags.Append(loadDiags...)
	if diags.HasError() {
		return nil, nil
	}

	service := r.newComposeService(ctx, diags)
	if diags.HasError() {
		return nil, nil
	}

	return project, service
}

func (r *dockerComposeResource) newComposeService(ctx context.Context, diags *diag.Diagnostics) composeapi.Service {
	if r.providerConfig == nil {
		diags.AddError("Provider not configured", "The provider configuration is unavailable for docker_compose resource operations.")
		return nil
	}

	client, err := r.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		diags.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return nil
	}

	dockerCli, err := createAndInitDockerCli(client)
	if err != nil {
		diags.AddError("Docker CLI error", err.Error())
		return nil
	}

	return compose.NewComposeService(dockerCli)
}

func buildComposeUpOptions(model dockerComposeResourceModel, project *composetypes.Project, diags *diag.Diagnostics) (composeapi.UpOptions, bool) {
	waitTimeout, ok := parseDurationAttribute("wait_timeout", model.WaitTimeout, diags)
	if !ok {
		return composeapi.UpOptions{}, false
	}

	return composeapi.UpOptions{
		Create: composeapi.CreateOptions{
			RemoveOrphans: model.RemoveOrphans.ValueBool(),
		},
		Start: composeapi.StartOptions{
			Project:     project,
			Wait:        model.Wait.ValueBool(),
			WaitTimeout: waitTimeout,
		},
	}, true
}

func loadComposeProject(ctx context.Context, model dockerComposeResourceModel) (*composetypes.Project, diag.Diagnostics) {
	var diags diag.Diagnostics

	configPaths, attrDiags := listToStrings(ctx, model.ConfigPaths)
	diags.Append(attrDiags...)
	if diags.HasError() {
		return nil, diags
	}

	if len(configPaths) == 0 {
		diags.AddError("Invalid compose files", "Attribute `config_paths` must contain at least one Compose file path.")
		return nil, diags
	}

	profiles, attrDiags := listToStrings(ctx, model.Profiles)
	diags.Append(attrDiags...)
	if diags.HasError() {
		return nil, diags
	}

	envFiles, attrDiags := listToStrings(ctx, model.EnvFiles)
	diags.Append(attrDiags...)
	if diags.HasError() {
		return nil, diags
	}

	projectOptionsFns := []composecli.ProjectOptionsFn{
		composecli.WithWorkingDirectory(strings.TrimSpace(model.ProjectDirectory.ValueString())),
		composecli.WithOsEnv,
		composecli.WithEnvFiles(envFiles...),
		composecli.WithDotEnv,
		composecli.WithDefaultProfiles(profiles...),
		composecli.WithResolvedPaths(true),
	}

	if projectName := strings.TrimSpace(model.ProjectName.ValueString()); projectName != "" {
		projectOptionsFns = append(projectOptionsFns, composecli.WithName(projectName))
	}

	projectOptions, err := composecli.NewProjectOptions(configPaths, projectOptionsFns...)
	if err != nil {
		diags.AddError("Invalid Compose options", err.Error())
		return nil, diags
	}

	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		diags.AddError("Docker Compose project load failed", err.Error())
		return nil, diags
	}

	for serviceName, service := range project.Services {
		service.CustomLabels = map[string]string{
			composeapi.ProjectLabel:     project.Name,
			composeapi.ServiceLabel:     serviceName,
			composeapi.VersionLabel:     composeapi.ComposeVersion,
			composeapi.WorkingDirLabel:  project.WorkingDir,
			composeapi.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			composeapi.OneoffLabel:      "False",
		}
		if len(envFiles) != 0 {
			service.CustomLabels[composeapi.EnvironmentFileLabel] = strings.Join(envFiles, ",")
		}
		project.Services[serviceName] = service
	}

	return project, diags
}

func listToStrings(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var values []string
	diags := value.ElementsAs(ctx, &values, false)
	return values, diags
}

func parseDurationAttribute(name string, value types.String, diags *diag.Diagnostics) (time.Duration, bool) {
	if value.IsNull() || value.IsUnknown() {
		return 0, true
	}

	rawValue := strings.TrimSpace(value.ValueString())
	if rawValue == "" {
		return 0, true
	}

	duration, err := time.ParseDuration(rawValue)
	if err != nil {
		diags.AddError("Invalid duration", fmt.Sprintf("Attribute `%s` must be a valid duration such as `30s` or `2m`: %s", name, err))
		return 0, false
	}

	return duration, true
}
