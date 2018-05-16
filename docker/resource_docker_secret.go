package docker

import (
	"encoding/base64"
	"log"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerSecretCreate,
		Read:   resourceDockerSecretRead,
		Delete: resourceDockerSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "User-defined name of the secret",
				Required:    true,
				ForceNew:    true,
			},

			"data": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "User-defined name of the secret",
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},
		},
	}
}

func resourceDockerSecretCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	createSecretOpts := dc.CreateSecretOptions{
		SecretSpec: swarm.SecretSpec{
			Annotations: swarm.Annotations{
				Name: d.Get("name").(string),
			},
			Data: data,
		},
	}

	secret, err := client.CreateSecret(createSecretOpts)
	if err != nil {
		return err
	}

	d.SetId(secret.ID)

	return resourceDockerSecretRead(d, meta)
}

func resourceDockerSecretRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	secret, err := client.InspectSecret(d.Id())

	if err != nil {
		if _, ok := err.(*dc.NoSuchSecret); ok {
			log.Printf("[WARN] Secret (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	d.SetId(secret.ID)
	return nil
}

func resourceDockerSecretDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	err := client.RemoveSecret(dc.RemoveSecretOptions{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
