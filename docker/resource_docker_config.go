package docker

import (
	"encoding/base64"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"

	"context"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDockerConfigCreate,
		ReadContext:   resourceDockerConfigRead,
		DeleteContext: resourceDockerConfigDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "User-defined name of the config",
				Required:    true,
				ForceNew:    true,
			},

			"data": {
				Type:         schema.TypeString,
				Description:  "Base64-url-safe-encoded config data",
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},
		},
	}
}

func resourceDockerConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	configSpec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		Data: data,
	}

	config, err := client.ConfigCreate(ctx, configSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(config.ID)

	return resourceDockerConfigRead(nil, d, meta)
}

func resourceDockerConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	config, _, err := client.ConfigInspectWithRaw(ctx, d.Id())

	if err != nil {
		log.Printf("[WARN] Config (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	d.SetId(config.ID)
	d.Set("name", config.Spec.Name)
	d.Set("data", base64.StdEncoding.EncodeToString(config.Spec.Data))
	return nil
}

func resourceDockerConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	err := client.ConfigRemove(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
