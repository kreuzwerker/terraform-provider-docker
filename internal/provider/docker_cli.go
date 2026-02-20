package provider

import (
	"fmt"
	"log"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
)

func createAndInitDockerCli(client *client.Client) (*command.DockerCli, error) {
	dockerCli, error := command.NewDockerCli()
	if error != nil {
		return nil, fmt.Errorf("failed to create Docker CLI: %w", error)
	}

	log.Printf("[DEBUG] Docker CLI initialized %#v, %#v", client, client.DaemonHost())
	err := dockerCli.Initialize(&flags.ClientOptions{Hosts: []string{client.DaemonHost()}})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker CLI: %w", err)
	}
	return dockerCli, nil
}
