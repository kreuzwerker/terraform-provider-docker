package docker

import (
	"fmt"
	"strings"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerImageCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	apiImage, err := findImage(d, client, meta.(*ProviderConfig).AuthConfigs)
	if err != nil {
		return fmt.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.SetId(apiImage.ID + d.Get("name").(string))
	d.Set("latest", apiImage.ID)

	return nil
}

func resourceDockerImageRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	var data Data
	if err := fetchLocalImages(&data, client); err != nil {
		return fmt.Errorf("Error reading docker image list: %s", err)
	}
	foundImage := searchLocalImages(data, d.Get("name").(string))

	if foundImage != nil {
		d.Set("latest", foundImage.ID)
	} else {
		d.SetId("")
	}

	return nil
}

func resourceDockerImageUpdate(d *schema.ResourceData, meta interface{}) error {
	// We need to re-read in case switching parameters affects
	// the value of "latest" or others
	client := meta.(*ProviderConfig).DockerClient
	apiImage, err := findImage(d, client, meta.(*ProviderConfig).AuthConfigs)
	if err != nil {
		return fmt.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.Set("latest", apiImage.ID)

	return nil
}

func resourceDockerImageDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	err := removeImage(d, client)
	if err != nil {
		return fmt.Errorf("Unable to remove Docker image: %s", err)
	}
	d.SetId("")
	return nil
}

func searchLocalImages(data Data, imageName string) *dc.APIImages {
	if apiImage, ok := data.DockerImages[imageName]; ok {
		return apiImage
	}
	if apiImage, ok := data.DockerImages[imageName+":latest"]; ok {
		imageName = imageName + ":latest"
		return apiImage
	}
	return nil
}

func removeImage(d *schema.ResourceData, client *dc.Client) error {
	var data Data

	if keepLocally := d.Get("keep_locally").(bool); keepLocally {
		return nil
	}

	if err := fetchLocalImages(&data, client); err != nil {
		return err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return fmt.Errorf("Empty image name is not allowed")
	}

	foundImage := searchLocalImages(data, imageName)

	if foundImage != nil {
		err := client.RemoveImage(foundImage.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchLocalImages(data *Data, client *dc.Client) error {
	images, err := client.ListImages(dc.ListImagesOptions{All: false})
	if err != nil {
		return fmt.Errorf("Unable to list Docker images: %s", err)
	}

	if data.DockerImages == nil {
		data.DockerImages = make(map[string]*dc.APIImages)
	}

	// Docker uses different nomenclatures in different places...sometimes a short
	// ID, sometimes long, etc. So we store both in the map so we can always find
	// the same image object. We store the tags, too.
	for i, image := range images {
		data.DockerImages[image.ID[:12]] = &images[i]
		data.DockerImages[image.ID] = &images[i]
		for _, repotag := range image.RepoTags {
			data.DockerImages[repotag] = &images[i]
		}
	}

	return nil
}

func pullImage(data *Data, client *dc.Client, authConfig *dc.AuthConfigurations, image string) error {
	// TODO: Test local registry handling. It should be working
	// based on the code that was ported over

	pullOpts := parseImageOptions(image)

	// If a registry was specified in the image name, try to find auth for it
	auth := dc.AuthConfiguration{}
	if pullOpts.Registry != "" {
		if authConfig, ok := authConfig.Configs[normalizeRegistryAddress(pullOpts.Registry)]; ok {
			auth = authConfig
		}
	} else {
		// Try to find an auth config for the public docker hub if a registry wasn't given
		if authConfig, ok := authConfig.Configs["https://registry.hub.docker.com"]; ok {
			auth = authConfig
		}
	}

	if err := client.PullImage(pullOpts, auth); err != nil {
		return fmt.Errorf("Error pulling image %s: %s\n", image, err)
	}

	return fetchLocalImages(data, client)
}

func parseImageOptions(image string) dc.PullImageOptions {
	pullOpts := dc.PullImageOptions{}

	// Pre-fill with image by default, update later if tag found
	pullOpts.Repository = image

	firstSlash := strings.Index(image, "/")

	// Detect the registry name - it should either contain port, be fully qualified or be localhost
	// If the image contains more than 2 path components, or at least one and the prefix looks like a hostname
	if strings.Count(image, "/") > 1 || firstSlash != -1 && (strings.ContainsAny(image[:firstSlash], ".:") || image[:firstSlash] == "localhost") {
		// registry/repo/image
		pullOpts.Registry = image[:firstSlash]
	}

	prefixLength := len(pullOpts.Registry)
	tagIndex := strings.Index(image[prefixLength:], ":")

	if tagIndex != -1 {
		// we have the tag, strip it
		pullOpts.Repository = image[:prefixLength+tagIndex]
		pullOpts.Tag = image[prefixLength+tagIndex+1:]
	}

	return pullOpts
}

func findImage(d *schema.ResourceData, client *dc.Client, authConfig *dc.AuthConfigurations) (*dc.APIImages, error) {
	var data Data
	if err := fetchLocalImages(&data, client); err != nil {
		return nil, err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return nil, fmt.Errorf("Empty image name is not allowed")
	}

	if err := pullImage(&data, client, authConfig, imageName); err != nil {
		return nil, fmt.Errorf("Unable to pull image %s: %s", imageName, err)
	}

	foundImage := searchLocalImages(data, imageName)
	if foundImage != nil {
		return foundImage, nil
	}

	return nil, fmt.Errorf("Unable to find or pull image %s", imageName)
}
