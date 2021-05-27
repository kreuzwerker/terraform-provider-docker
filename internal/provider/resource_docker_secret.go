package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerSecret() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the secrets of a Docker service in a swarm.",

		CreateContext: resourceDockerSecretCreate,
		ReadContext:   resourceDockerSecretRead,
		DeleteContext: resourceDockerSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "User-defined name of the secret",
				Required:    true,
				ForceNew:    true,
			},

			"data": {
				Type:             schema.TypeString,
				Description:      "Base64-url-safe-encoded secret data",
				Required:         true,
				Sensitive:        true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringIsBase64Encoded(),
			},

			"labels": {
				Type:        schema.TypeSet,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
				Elem:        labelSchema,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDockerSecretV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					return replaceLabelsMapFieldWithSetField(rawState), nil
				},
			},
		},
	}
}

func resourceDockerSecretV0() *schema.Resource {
	return &schema.Resource{
		// This is only used for state migration, so the CRUD
		// callbacks are no longer relevant
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "User-defined name of the secret",
				Required:    true,
				ForceNew:    true,
			},

			"data": {
				Type:             schema.TypeString,
				Description:      "User-defined name of the secret",
				Required:         true,
				Sensitive:        true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringIsBase64Encoded(),
			},

			"labels": {
				Type:        schema.TypeMap,
				Description: "User-defined key/value metadata",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceDockerSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	secretSpec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		Data: data,
	}

	if v, ok := d.GetOk("labels"); ok {
		secretSpec.Annotations.Labels = labelSetToMap(v.(*schema.Set))
	}

	secret, err := client.SecretCreate(ctx, secretSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(secret.ID)

	return resourceDockerSecretRead(ctx, d, meta)
}

func resourceDockerSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	secret, _, err := client.SecretInspectWithRaw(ctx, d.Id())
	if err != nil {
		log.Printf("[WARN] Secret (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	jsonObj, _ := json.MarshalIndent(secret, "", "\t")
	log.Printf("[DEBUG] Docker secret inspect from readFunc: %s", jsonObj)

	d.SetId(secret.ID)
	d.Set("name", secret.Spec.Name)
	// Note mavogel: secret data is not exposed via the API
	// TODO next major if we do not explicitly store it in the state we could import it, but BC
	// d.Set("data", base64.StdEncoding.EncodeToString(secret.Spec.Data))
	return nil
}

func resourceDockerSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	err := client.SecretRemove(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
