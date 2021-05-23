package provider

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resourceType the type of the resource
type resourceType int

const (
	RESOURCE    resourceType = iota // a resource
	DATA_SOURCE                     // a data-source
)

// String converts the the resourceType into
// the name of the directory the test configuartions
// are int
func (r resourceType) String() string {
	return [...]string{
		"resources",
		"data-sources",
	}[r]
}

const (
	TEST_CONFIG_BASE_DIR = "testdata"
	TEST_CONFIG_FILENAME = "test-config.tf"
)

// loadTestConfiguration loads the configuration for the test for the type of the
// resource, the resource itself, like 'docker_container' and the name of the test,
// like 'testAccDockerContainerPrivateImage'
//
// As a convention the test configurations are in
// 'testdata/<resourceType>/<resourceName>/<testName>/test-config.tf', e.g.
// 'testdata/resources/docker_container/testAccDockerContainerPrivateImage/test-config.tf'
//
func loadTestConfiguration(t *testing.T, resourceType resourceType, resourceName, testName string) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory: %w", err)
	}

	testConfig := strings.ReplaceAll(filepath.Join(wd, "..", "..", TEST_CONFIG_BASE_DIR, resourceType.String(), resourceName, testName, TEST_CONFIG_FILENAME), "\\", "\\\\")

	testConfigContent, err := ioutil.ReadFile(testConfig)
	if err != nil {
		t.Errorf("failed to read test configuration at '%s': %w", testConfig, err)
	}

	return string(testConfigContent)
}
