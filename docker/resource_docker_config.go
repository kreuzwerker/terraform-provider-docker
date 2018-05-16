package docker

import (
	"encoding/base64"
	"log"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerConfigCreate,
		Read:   resourceDockerConfigRead,
		Delete: resourceDockerConfigDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "User-defined name of the config",
				Required:    true,
				ForceNew:    true,
			},

			"data": &schema.Schema{
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

	createConfigOpts := dc.CreateConfigOptions{
		ConfigSpec: swarm.ConfigSpec{
			Annotations: swarm.Annotations{
				Name: d.Get("name").(string),
			},
			Data: data,
		},
	}

	config, err := client.CreateConfig(createConfigOpts)
	if err != nil {
		return err
	}
	d.SetId(config.ID)

	return resourceDockerConfigRead(d, meta)
}

func resourceDockerConfigRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	config, err := client.InspectConfig(d.Id())

	if err != nil {
		if _, ok := err.(*dc.NoSuchConfig); ok {
			log.Printf("[WARN] Config (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	d.SetId(config.ID)
	return nil
}

func resourceDockerConfigDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	err := client.RemoveConfig(dc.RemoveConfigOptions{
		ID: d.Id(),
	})
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
