package actiontests

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	providerpkg "github.com/terraform-providers/terraform-provider-docker/internal/provider"
)

func TestDockerExecAction_createsFileInBusyboxContainer(t *testing.T) {
	preCheckDocker(t)

	containerName := fmt.Sprintf("tf-acc-docker-exec-%d", time.Now().UnixNano())
	filePath := "/tmp/docker_exec_action_file"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "docker_container" "target" {
  name     = %q
  image    = docker_image.busybox.image_id
  must_run = true
  command  = ["sh", "-c", "sleep 300"]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_docker_exec.create_file]
    }
  }
}

action "docker_docker_exec" "create_file" {
  config {
    container = docker_container.target.name
    command   = ["sh", "-c", "touch %s"]
  }
}
`, containerName, filePath),
				PostApplyFunc: func() {
					cmd := exec.Command("docker", "exec", containerName, "sh", "-c", "test -f "+filePath)
					if err := cmd.Run(); err != nil {
						t.Fatalf("expected file %q to exist in container %q: %s", filePath, containerName, err)
					}
				},
			},
		},
	})
}

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"docker": func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()

			upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, providerpkg.New("test")().GRPCProvider)
			if err != nil {
				return nil, err
			}

			frameworkProviderServerFactory := providerserver.NewProtocol6WithError(providerpkg.NewFrameworkProvider("test")())

			muxServer, err := tf6muxserver.NewMuxServer(
				ctx,
				func() tfprotov6.ProviderServer {
					return upgradedSDKProvider
				},
				func() tfprotov6.ProviderServer {
					frameworkProviderServer, frameworkErr := frameworkProviderServerFactory()
					if frameworkErr != nil {
						panic(frameworkErr)
					}

					return frameworkProviderServer
				},
			)
			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
		},
	}
}

func preCheckDocker(t *testing.T) {
	t.Helper()
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Docker must be available: %s", err)
	}
}
