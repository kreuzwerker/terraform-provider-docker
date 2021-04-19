package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func resourceDockerVolumeV0() *schema.Resource {
	return &schema.Resource{
		// This is only used for state migration, so the CRUD
		// callbacks are no longer relevant
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"mountpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
