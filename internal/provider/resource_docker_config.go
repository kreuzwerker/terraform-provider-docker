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
				Optional:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"data", "data_raw"},
				ValidateDiagFunc: validateStringIsBase64Encoded(),
			},
			"data_raw": {
				Type:         schema.TypeString,
				Description:  "Raw (plain text) config data",
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"data", "data_raw"},
			},
			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
				Elem:        labelSchema,
			},
		},
	}
}

func resourceDockerConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	data, err := getConfigDataBytes(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to decode config data: %w", err))
	}

	configSpec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		Data: data,
	}

	if v, ok := d.GetOk("labels"); ok {
		configSpec.Labels = labelSetToMap(v.(*schema.Set))
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
	if err := d.Set("name", config.Spec.Name); err != nil {
		return diag.FromErr(err)
	}
	if _, hasDataRaw := d.GetOk("data_raw"); hasDataRaw {
		if err := d.Set("data_raw", string(config.Spec.Data)); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("data", base64.StdEncoding.EncodeToString(config.Spec.Data)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("labels", mapToLabelSet(config.Spec.Labels)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// getConfigDataBytes returns config payload bytes from data_raw when present, or
// from base64-decoded data otherwise.
func getConfigDataBytes(d *schema.ResourceData) ([]byte, error) {
	if dataRaw, ok := d.GetOk("data_raw"); ok {
		return []byte(dataRaw.(string)), nil
	}

	data, err := base64.StdEncoding.DecodeString(d.Get("data").(string))
	if err != nil {
		return nil, err
	}

	return data, nil
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
