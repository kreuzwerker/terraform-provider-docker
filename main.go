package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-docker/docker"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: docker.Provider})
}
