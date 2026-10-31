package provider

import (
	"fmt"
	"log"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
)

// createAndInitDockerCli returns a Docker CLI bound to the given API client.
//
// configuredHost is the provider's `host` value. It is also set as the CLI's
// --host option: buildx (and compose) resolve the "default" context endpoint
// from the CLI options and open their own connections to it, so without it a
// build would go to DOCKER_HOST or the local socket instead of the configured
// remote daemon. The configured host is preferred over client.DaemonHost()
// because the latter is rewritten for ssh:// hosts.
func createAndInitDockerCli(client *client.Client, configuredHost string) (*command.DockerCli, error) {
	dockerCli, error := command.NewDockerCli(command.WithAPIClient(client))
	if error != nil {
		return nil, fmt.Errorf("failed to create Docker CLI: %w", error)
	}

	log.Printf("[DEBUG] Docker CLI initialized %#v", client)
	options := flags.NewClientOptions()
	if configuredHost != "" {
		options.Hosts = []string{configuredHost}
	} else if client.DaemonHost() != "" {
		options.Hosts = []string{client.DaemonHost()}
	}

	err := dockerCli.Initialize(options, command.WithAPIClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker CLI: %w", err)
	}
	return dockerCli, nil
}
