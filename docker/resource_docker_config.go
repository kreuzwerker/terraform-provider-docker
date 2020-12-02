package docker

import (
	"context"
	"encoding/base64"
	"log"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerConfigCreate,
		Read:   resourceDockerConfigRead,
		Delete: resourceDockerConfigDelete,
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

func resourceDockerConfigCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	configSpec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		Data: data,
	}

	config, err := client.ConfigCreate(context.Background(), configSpec)
	if err != nil {
		return err
	}
	d.SetId(config.ID)

	return resourceDockerConfigRead(d, meta)
}

func resourceDockerConfigRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	config, _, err := client.ConfigInspectWithRaw(context.Background(), d.Id())
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

func resourceDockerConfigDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	err := client.ConfigRemove(context.Background(), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
