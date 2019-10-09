package docker

import (
	"encoding/base64"
	"log"

	"context"
	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerSecretCreate,
		Read:   resourceDockerSecretRead,
		Delete: resourceDockerSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "User-defined name of the secret",
				Required:    true,
				ForceNew:    true,
			},

			"data": {
				Type:         schema.TypeString,
				Description:  "User-defined name of the secret",
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDockerSecretCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	secretSpec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		Data: data,
	}

	if v, ok := d.GetOk("labels"); ok {
		secretSpec.Annotations.Labels = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	secret, err := client.SecretCreate(context.Background(), secretSpec)
	if err != nil {
		return err
	}

	d.SetId(secret.ID)

	return resourceDockerSecretRead(d, meta)
}

func resourceDockerSecretRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	secret, _, err := client.SecretInspectWithRaw(context.Background(), d.Id())

	if err != nil {
		log.Printf("[WARN] Secret (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	d.SetId(secret.ID)
	return nil
}

func resourceDockerSecretDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	err := client.SecretRemove(context.Background(), d.Id())

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
