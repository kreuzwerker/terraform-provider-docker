package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDockerLogsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:             loadTestConfiguration(t, DATA_SOURCE, "docker_logs", "testAccDockerLogsDataSourceBasic"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_logs.logs_discard_headers_false", "logs_list_string.0", "\u0001\u0000\u0000\u0000\u0000\u0000\u0000\fHello World"),
					resource.TestCheckResourceAttr("data.docker_logs.logs_discard_headers_false", "logs_list_string.1", "\u0001\u0000\u0000\u0000\u0000\u0000\u0000\fHello World"),
					resource.TestCheckResourceAttr("data.docker_logs.logs_discard_headers_false", "logs_list_string.2", "\u0001\u0000\u0000\u0000\u0000\u0000\u0000\fHello World"),
					resource.TestCheckResourceAttr("data.docker_logs.logs_basic", "logs_list_string.0", "Hello World"),
					resource.TestCheckResourceAttr("data.docker_logs.logs_basic", "logs_list_string.1", "Hello World"),
					resource.TestCheckResourceAttr("data.docker_logs.logs_basic", "logs_list_string.2", "Hello World"),
				),
			},
		},
	})
}
