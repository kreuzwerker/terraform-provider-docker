package provider

import (
	"strings"

	"github.com/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func parseOptionalPlatform(value string) (*specs.Platform, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	platformValue, err := platforms.Parse(trimmed)
	if err != nil {
		return nil, err
	}

	return &platformValue, nil
}
