//go:build !no_compose

package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

// composeResources returns the resources backed by github.com/docker/compose/v2.
// These resources transitively pull in docker/compose/v2/pkg/watch ->
// github.com/fsnotify/fsevents on darwin, which requires CGO. Consumers
// cross-compiling without CGO can exclude them with the no_compose build tag.
func composeResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewDockerComposeResource,
	}
}
