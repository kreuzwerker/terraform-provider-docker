package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DockerExecAction struct {
	providerConfig *ProviderConfig
}

type DockerExecActionModel struct {
	Container  types.String `tfsdk:"container"`
	Command    types.List   `tfsdk:"command"`
	Detach     types.Bool   `tfsdk:"detach"`
	Env        types.List   `tfsdk:"env"`
	EnvFile    types.List   `tfsdk:"env_file"`
	Privileged types.Bool   `tfsdk:"privileged"`
	TTY        types.Bool   `tfsdk:"tty"`
	User       types.String `tfsdk:"user"`
	Workdir    types.String `tfsdk:"workdir"`
}

// The action implementation
func (a *DockerExecAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exec"
}

func (a *DockerExecAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Run a command in an existing container, similar to `docker container exec`. Please note that due to the nature of actions, we cannot have an computed `output` attribute that contains the command output.",
		Attributes: map[string]actionschema.Attribute{
			"container": actionschema.StringAttribute{
				MarkdownDescription: "Container name or ID where the command is executed.",
				Required:            true,
			},
			"command": actionschema.ListAttribute{
				MarkdownDescription: "Command and arguments to execute inside the container.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"detach": actionschema.BoolAttribute{
				MarkdownDescription: "Run command in the background.",
				Optional:            true,
			},
			"env": actionschema.ListAttribute{
				MarkdownDescription: "Set environment variables for the command (`KEY=value` or `KEY`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"env_file": actionschema.ListAttribute{
				MarkdownDescription: "Read in environment variables from files.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"privileged": actionschema.BoolAttribute{
				MarkdownDescription: "Give extended privileges to the command.",
				Optional:            true,
			},
			"tty": actionschema.BoolAttribute{
				MarkdownDescription: "Allocate a pseudo-TTY.",
				Optional:            true,
			},
			"user": actionschema.StringAttribute{
				MarkdownDescription: "Username or UID (format: `<name|uid>[:<group|gid>]`).",
				Optional:            true,
			},
			"workdir": actionschema.StringAttribute{
				MarkdownDescription: "Working directory inside the container.",
				Optional:            true,
			},
		},
	}
}

func (a *DockerExecAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DockerExecAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.providerConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider configuration is unavailable for docker exec action invocation.")
		return
	}

	var config DockerExecActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var command []string
	resp.Diagnostics.Append(config.Command.ElementsAs(ctx, &command, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(command) == 0 {
		resp.Diagnostics.AddError("Invalid command", "Attribute `command` must contain at least one element.")
		return
	}

	environment, err := getExecEnvironment(ctx, &config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment configuration", err.Error())
		return
	}

	client, err := a.providerConfig.MakeClient(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Docker client error", fmt.Sprintf("Unable to create Docker client: %s", err))
		return
	}

	detach := config.Detach.ValueBool()
	tty := config.TTY.ValueBool()

	execOptions := container.ExecOptions{
		User:         config.User.ValueString(),
		Privileged:   config.Privileged.ValueBool(),
		Tty:          tty,
		AttachStdin:  false,
		AttachStdout: !detach,
		AttachStderr: !detach,
		DetachKeys:   "", // Does not make sense to support custom detach keys in Terraform, as the action will not be interactive and will not be able to listen for key sequences. The command can be detached by setting `detach=true`.
		Env:          environment,
		WorkingDir:   config.Workdir.ValueString(),
		Cmd:          command,
	}

	execCreateResponse, err := client.ContainerExecCreate(ctx, config.Container.ValueString(), execOptions)
	if err != nil {
		resp.Diagnostics.AddError("Docker exec create failed", err.Error())
		return
	}

	if detach {
		resp.Diagnostics.AddWarning(
			"Detached exec is not tracked",
			"The command is started in background mode and Terraform will not stream output or track completion/exit code after invocation.",
		)

		err = client.ContainerExecStart(ctx, execCreateResponse.ID, container.ExecStartOptions{Detach: true, Tty: tty})
		if err != nil {
			resp.Diagnostics.AddError("Docker exec start failed", err.Error())
			return
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Command started in detached mode."})
		}
		return
	}

	attachResponse, err := client.ContainerExecAttach(ctx, execCreateResponse.ID, container.ExecAttachOptions{Tty: tty})
	if err != nil {
		resp.Diagnostics.AddError("Docker exec attach failed", err.Error())
		return
	}
	defer attachResponse.Close() // nolint:errcheck

	var stdout, stderr bytes.Buffer
	if tty {
		_, err = io.Copy(&stdout, attachResponse.Reader)
	} else {
		_, err = stdcopy.StdCopy(&stdout, &stderr, attachResponse.Reader)
	}
	if err != nil {
		resp.Diagnostics.AddError("Docker exec output error", err.Error())
		return
	}

	inspect, err := client.ContainerExecInspect(ctx, execCreateResponse.ID)
	if err != nil {
		resp.Diagnostics.AddError("Docker exec inspect failed", err.Error())
		return
	}

	if resp.SendProgress != nil {
		if out := strings.TrimSpace(stdout.String()); out != "" {
			resp.SendProgress(action.InvokeProgressEvent{Message: out})
		}
		if errOut := strings.TrimSpace(stderr.String()); errOut != "" {
			resp.SendProgress(action.InvokeProgressEvent{Message: errOut})
		}
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("exit_code=%d", inspect.ExitCode)})
	}

	if inspect.ExitCode != 0 {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}

		resp.Diagnostics.AddError(
			fmt.Sprintf("Docker exec returned non-zero exit code: %d", inspect.ExitCode),
			details,
		)
		return
	}
}

func getExecEnvironment(ctx context.Context, config *DockerExecActionModel) ([]string, error) {
	result := make([]string, 0)

	if !config.Env.IsNull() && !config.Env.IsUnknown() {
		var inlineEnv []string
		if diags := config.Env.ElementsAs(ctx, &inlineEnv, false); diags.HasError() {
			return nil, fmt.Errorf("failed parsing `env`: %s", diags.Errors())
		}

		result = append(result, resolveEnvironmentVariables(inlineEnv)...)
	}

	if !config.EnvFile.IsNull() && !config.EnvFile.IsUnknown() {
		var envFiles []string
		if diags := config.EnvFile.ElementsAs(ctx, &envFiles, false); diags.HasError() {
			return nil, fmt.Errorf("failed parsing `env_file`: %s", diags.Errors())
		}

		for _, envFile := range envFiles {
			fileEnvs, err := parseEnvFile(envFile)
			if err != nil {
				return nil, fmt.Errorf("failed parsing env_file %q: %w", envFile, err)
			}

			result = append(result, resolveEnvironmentVariables(fileEnvs)...)
		}
	}

	return result, nil
}

func parseEnvFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint:errcheck

	result := make([]string, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		result = append(result, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func resolveEnvironmentVariables(items []string) []string {
	result := make([]string, 0, len(items))

	for _, item := range items {
		if strings.Contains(item, "=") {
			result = append(result, item)
			continue
		}

		if value, ok := os.LookupEnv(item); ok {
			result = append(result, fmt.Sprintf("%s=%s", item, value))
			continue
		}

		result = append(result, fmt.Sprintf("%s=", item))
	}

	return result
}
