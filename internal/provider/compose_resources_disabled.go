//go:build no_compose

package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

// composeResources returns no resources when the provider is built with the
// no_compose build tag. The docker_compose resource pulls in
// docker/compose/v2/pkg/watch -> github.com/fsnotify/fsevents on darwin, which
// requires CGO at build time; consumers that cross-compile without CGO use
// this tag to keep the rest of the provider buildable.
func composeResources() []func() resource.Resource {
	return nil
}
