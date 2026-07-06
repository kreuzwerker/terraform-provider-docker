package provider

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestComposeVersionLabelFallsBackToBuildInfo(t *testing.T) {
	got := composeVersionLabelFor("", debug.ReadBuildInfo)
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
