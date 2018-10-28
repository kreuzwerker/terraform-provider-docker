package docker

import (
	"context"
	"fmt"
	"log"
	"strings"

	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

	return resourceDockerImageRead(d, meta)
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

	return resourceDockerImageRead(d, meta)
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

func searchLocalImages(data Data, imageName string) *types.ImageSummary {
	if apiImage, ok := data.DockerImages[imageName]; ok {
		return apiImage
	}
	if apiImage, ok := data.DockerImages[imageName+":latest"]; ok {
		imageName = imageName + ":latest"
		return apiImage
	}
	return nil
}

func removeImage(d *schema.ResourceData, client *client.Client) error {
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
		imageDeleteResponseItems, err := client.ImageRemove(context.Background(), foundImage.ID, types.ImageRemoveOptions{})
		if err != nil {
			return err
		}
		log.Printf("[INFO] Deleted image items: %v", imageDeleteResponseItems)
	}

	return nil
}

func fetchLocalImages(data *Data, client *client.Client) error {
	images, err := client.ImageList(context.Background(), types.ImageListOptions{All: false})
	if err != nil {
		return fmt.Errorf("Unable to list Docker images: %s", err)
	}

	if data.DockerImages == nil {
		data.DockerImages = make(map[string]*types.ImageSummary)
	}

	// Docker uses different nomenclatures in different places...sometimes a short
	// ID, sometimes long, etc. So we store both in the map so we can always find
	// the same image object. We store the tags and digests, too.
	for i, image := range images {
		data.DockerImages[image.ID[:12]] = &images[i]
		data.DockerImages[image.ID] = &images[i]
		for _, repotag := range image.RepoTags {
			data.DockerImages[repotag] = &images[i]
		}
		for _, repodigest := range image.RepoDigests {
			data.DockerImages[repodigest] = &images[i]
		}
	}

	return nil
}

func pullImage(data *Data, client *client.Client, authConfig *AuthConfigs, image string) error {
	pullOpts := parseImageOptions(image)

	// If a registry was specified in the image name, try to find auth for it
	auth := types.AuthConfig{}
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

	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("error creating auth config: %s", err)
	}

	out, err := client.ImagePull(context.Background(), image, types.ImagePullOptions{
		RegistryAuth: base64.URLEncoding.EncodeToString(encodedJSON),
	})
	if err != nil {
		return fmt.Errorf("error pulling image %s: %s", image, err)
	}
	defer out.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)
	s := buf.String()
	log.Printf("[DEBUG] pulled image %v: %v", image, s)

	return fetchLocalImages(data, client)
}

type internalPullImageOptions struct {
	Repository string `qs:"fromImage"`
	Tag        string

	// Only required for Docker Engine 1.9 or 1.10 w/ Remote API < 1.21
	// and Docker Engine < 1.9
	// This parameter was removed in Docker Engine 1.11
	Registry string
}

func parseImageOptions(image string) internalPullImageOptions {
	pullOpts := internalPullImageOptions{}
	// splitImageName := strings.Split(image, ":")
	// switch len(splitImageName) {

	// // It's in registry:port/username/repo:tag or registry:port/repo:tag format
	// case 3:
	// 	splitPortRepo := strings.Split(splitImageName[1], "/")
	// 	pullOpts.Registry = splitImageName[0] + ":" + splitPortRepo[0]
	// 	pullOpts.Tag = splitImageName[2]
	// 	pullOpts.Repository = pullOpts.Registry + "/" + strings.Join(splitPortRepo[1:], "/")

	// // It's either registry:port/username/repo, registry:port/repo
	// // or repo:tag with default registry
	// case 2:
	// 	splitPortRepo := strings.Split(splitImageName[1], "/")
	// 	switch len(splitPortRepo) {
	// 	// repo:tag
	// 	case 1:
	// 		pullOpts.Repository = splitImageName[0]
	// 		pullOpts.Tag = splitImageName[1]

	// 	// registry:port/username/repo or registry:port/repo
	// 	default:
	// 		pullOpts.Registry = splitImageName[0] + ":" + splitPortRepo[0]
	// 		pullOpts.Repository = pullOpts.Registry + "/" + strings.Join(splitPortRepo[1:], "/")
	// 		pullOpts.Tag = "latest"
	// 	}

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

func findImage(d *schema.ResourceData, client *client.Client, authConfig *AuthConfigs) (*types.ImageSummary, error) {
	var data Data
	//if err := fetchLocalImages(&data, client); err != nil {
	//	return nil, err
	//} Is done in pullImage

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
