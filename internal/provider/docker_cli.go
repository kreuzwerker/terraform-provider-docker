package provider

import (
	"fmt"
	"log"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
)

func createAndInitDockerCli(client *client.Client) (*command.DockerCli, error) {
	dockerCli, error := command.NewDockerCli(command.WithAPIClient(client))
	if error != nil {
		return nil, fmt.Errorf("failed to create Docker CLI: %w", error)
	}

	log.Printf("[DEBUG] Docker CLI initialized %#v", client)
	options := flags.NewClientOptions()

	err := dockerCli.Initialize(options, command.WithAPIClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker CLI: %w", err)
	}
	return dockerCli, nil
}
