package docker

import (
	"github.com/fsouza/go-dockerclient"
	"testing"
)

func TestParseRegistryPortTagFormat(t *testing.T) {
	validatePullOpts(t,
		"registry.gitlab.com:443/foo/bar:v1",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo/bar",
			Tag:        "v1",
		})
	validatePullOpts(t,
		"registry.gitlab.com:443/foo:v1",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo",
			Tag:        "v1",
		})
}

func TestParseRegistryPortFormat(t *testing.T) {
	validatePullOpts(t,
		"registry.gitlab.com:443/foo/bar",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo/bar",
			Tag:        "",
		})
	validatePullOpts(t,
		"registry.gitlab.com:443/foo",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo",
			Tag:        "",
		})
}

func TestParseRepoTagFormat(t *testing.T) {
	validatePullOpts(t,
		"foo:bar",
		&docker.PullImageOptions{
			Registry:   "",
			Repository: "foo",
			Tag:        "bar",
		})
}

func TestParseRegistryRepoFormat(t *testing.T) {
	validatePullOpts(t,
		"registry.gitlab.com/foo/bar",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com",
			Repository: "registry.gitlab.com/foo/bar",
			Tag:        "",
		})
}

func TestParsePlainRepoFormat(t *testing.T) {
	validatePullOpts(t,
		"foo/bar",
		&docker.PullImageOptions{
			Registry:   "",
			Repository: "foo/bar",
			Tag:        "",
		})
}

func TestParseGitlabComThreePartImageOptions(t *testing.T) {
	validatePullOpts(t,
		"registry.gitlab.com:443/foo/bar/baz:v1",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo/bar/baz",
			Tag:        "v1",
		})
	validatePullOpts(t,
		"registry.gitlab.com:443/foo/bar/baz",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com:443",
			Repository: "registry.gitlab.com:443/foo/bar/baz",
			Tag:        "",
		})
	validatePullOpts(t,
		"registry.gitlab.com/foo/bar/baz:v1",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com",
			Repository: "registry.gitlab.com/foo/bar/baz",
			Tag:        "v1",
		})
	validatePullOpts(t,
		"registry.gitlab.com/foo/bar/baz",
		&docker.PullImageOptions{
			Registry:   "registry.gitlab.com",
			Repository: "registry.gitlab.com/foo/bar/baz",
			Tag:        "",
		})
}

func validatePullOpts(t *testing.T, inputString string, expected *docker.PullImageOptions) {
	pullOpts := parseImageOptions(inputString)

	if pullOpts.Registry != expected.Registry {
		t.Fatalf("For '%s' expected registry '%s', got '%s'", inputString, expected.Registry, pullOpts.Registry)
	}

	if pullOpts.Repository != expected.Repository {
		t.Fatalf("For '%s' expected repository '%s', got '%s'", inputString, expected.Repository, pullOpts.Repository)
	}

	if pullOpts.Tag != expected.Tag {
		t.Fatalf("For '%s' expected tag '%s', got '%s'", inputString, expected.Tag, pullOpts.Tag)
	}
}
