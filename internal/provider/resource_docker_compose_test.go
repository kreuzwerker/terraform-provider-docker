package provider

import (
	"runtime/debug"
	"strings"
	"testing"

	composeapi "github.com/docker/compose/v2/pkg/api"
)

func TestComposeVersionLabelFallsBackToBuildInfo(t *testing.T) {
	original := composeapi.ComposeVersion
	composeapi.ComposeVersion = ""
	t.Cleanup(func() {
		composeapi.ComposeVersion = original
	})

	got := composeVersionLabel()
	if got == "" {
		t.Fatal("compose version label must not be empty")
	}

	expected := "unknown"
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/docker/compose/v2" {
				expected = strings.TrimPrefix(dep.Version, "v")
				break
			}
		}
	}

	if got != expected {
		t.Fatalf("compose version label = %q, want %q", got, expected)
	}
}
