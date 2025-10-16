package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerConfig() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the configs of a Docker service in a swarm.",

		CreateContext: resourceDockerConfigCreate,
		ReadContext:   resourceDockerConfigRead,
		DeleteContext: resourceDockerConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "User-defined name of the config",
				Required:    true,
				ForceNew:    true,
			},

			"data": {
				Type:             schema.TypeString,
				Description:      "Base64-url-safe-encoded config data",
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringIsBase64Encoded(),
			},
		},
	}
}

func resourceDockerConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
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

	return resourceDockerConfigRead(ctx, d, meta)
}

func resourceDockerConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	config, _, err := client.ConfigInspectWithRaw(ctx, d.Id())
	if err != nil {
		log.Printf("[WARN] Config (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	jsonObj, _ := json.MarshalIndent(config, "", "\t")
	log.Printf("[DEBUG] Docker config inspect from readFunc: %s", jsonObj)

	d.SetId(config.ID)
	d.Set("name", config.Spec.Name)                                    //nolint:errcheck
	d.Set("data", base64.StdEncoding.EncodeToString(config.Spec.Data)) //nolint:errcheck
	return nil
}

func resourceDockerConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	err = client.ConfigRemove(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("") //nolint:errcheck
	return nil
}
