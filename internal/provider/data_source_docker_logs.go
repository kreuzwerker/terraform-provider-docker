package provider

import (
	"bufio"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerLogs() *schema.Resource {
	return &schema.Resource{
		Description: "`docker_logs` provides logs from specific container",

		ReadContext: dataSourceDockerLogsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker Container",
				Required:    true,
			},
			"logs_list_string": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"logs_list_string_enabled": {
				Type:        schema.TypeBool,
				Default:     true,
				Optional:    true,
				Description: "If true populate computed value `logs_list_string`",
			},
			"show_stdout": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},
			"show_stderr": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},
			"since": {
				Type:     schema.TypeString,
				Default:  "",
				Optional: true,
			},
			"until": {
				Type:     schema.TypeString,
				Default:  "",
				Optional: true,
			},
			"timestamps": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"follow": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"tail": {
				Type:     schema.TypeString,
				Default:  "",
				Optional: true,
			},
			"details": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func dataSourceDockerLogsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	container := d.Get("name").(string)
	d.SetId(container)

	// call client for logs
	readCloser, err := client.ContainerLogs(ctx, container, types.ContainerLogsOptions{
		ShowStdout: d.Get("show_stdout").(bool),
		ShowStderr: d.Get("show_stderr").(bool),
		Since:      d.Get("since").(string),
		Until:      d.Get("until").(string),
		Timestamps: d.Get("timestamps").(bool),
		Follow:     d.Get("follow").(bool),
		Tail:       d.Get("tail").(string),
		Details:    d.Get("details").(bool),
	})
	if err != nil {
		return diag.Errorf("dataSourceDockerLogsRead: error while asking for logs %s", err)
	}
	defer readCloser.Close()

	// read string lines
	if d.Get("logs_list_string_enabled").(bool) {
		lines := make([]string, 0)
		scanner := bufio.NewScanner(readCloser)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := d.Set("logs_list_string", lines); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
