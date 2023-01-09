package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
	"github.com/moby/buildkit/session"
	"github.com/pkg/errors"
)

const minBuildkitDockerVersion = "1.39"

func resourceDockerImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	imageName := d.Get("name").(string)

	if value, ok := d.GetOk("build"); ok {
		for _, rawBuild := range value.(*schema.Set).List() {
			rawBuild := rawBuild.(map[string]interface{})

			err := buildDockerImage(ctx, rawBuild, imageName, client)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	apiImage, err := findImage(ctx, imageName, client, meta.(*ProviderConfig).AuthConfigs, d.Get("platform").(string))
	if err != nil {
		return diag.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.SetId(apiImage.ID + d.Get("name").(string))
	return resourceDockerImageRead(ctx, d, meta)
}

func resourceDockerImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	var data Data
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return diag.Errorf("Error reading docker image list: %s", err)
	}
	for id := range data.DockerImages {
		log.Printf("[DEBUG] local images data: %v", id)
	}

	imageName := d.Get("name").(string)

	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return diag.Errorf("resourceDockerImageRead: error looking up local image %q: %s", imageName, err)
	}
	if foundImage == nil {
		log.Printf("[DEBUG] did not find image with name: %v", imageName)
		d.SetId("")
		return nil
	}

	repoDigest := determineRepoDigest(imageName, foundImage)

	// TODO mavogel: remove the appended name from the ID
	d.SetId(foundImage.ID + d.Get("name").(string))
	d.Set("image_id", foundImage.ID)
	d.Set("repo_digest", repoDigest)
	return nil
}

func resourceDockerImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// We need to re-read in case switching parameters affects
	// the value of "latest" or others
	client := meta.(*ProviderConfig).DockerClient
	imageName := d.Get("name").(string)
	_, err := findImage(ctx, imageName, client, meta.(*ProviderConfig).AuthConfigs, d.Get("platform").(string))
	if err != nil {
		return diag.Errorf("Unable to read Docker image into resource: %s", err)
	}

	return resourceDockerImageRead(ctx, d, meta)
}

func resourceDockerImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	// TODO mavogel: add retries. see e.g. service updateFailsAndRollbackConvergeConfig test
	err := removeImage(ctx, d, client)
	if err != nil {
		return diag.Errorf("Unable to remove Docker image: %s", err)
	}
	d.SetId("")
	return nil
}

// Helpers
func searchLocalImages(ctx context.Context, client *client.Client, data Data, imageName string) (*types.ImageSummary, error) {
	imageInspect, _, err := client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to inspect image %s: %w", imageName, err)
	}

	jsonObj, err := json.MarshalIndent(imageInspect, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("error parsing inspect response: %w", err)
	}
	log.Printf("[DEBUG] Docker image inspect from readFunc: %s", jsonObj)

	if apiImage, ok := data.DockerImages[imageInspect.ID]; ok {
		log.Printf("[DEBUG] found local image via imageName: %v", imageName)
		return apiImage, nil
	}
	return nil, nil
}

func removeImage(ctx context.Context, d *schema.ResourceData, client *client.Client) error {
	var data Data

	if keepLocally := d.Get("keep_locally").(bool); keepLocally {
		return nil
	}

	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return fmt.Errorf("empty image name is not allowed")
	}

	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return fmt.Errorf("removeImage: error looking up local image %q: %w", imageName, err)
	}

	if foundImage != nil {
		imageDeleteResponseItems, err := client.ImageRemove(ctx, imageName, types.ImageRemoveOptions{
			Force: d.Get("force_remove").(bool),
		})
		if err != nil {
			return err
		}
		indentedImageDeleteResponseItems, _ := json.MarshalIndent(imageDeleteResponseItems, "", "\t")
		log.Printf("[DEBUG] Deleted image items: \n%s", indentedImageDeleteResponseItems)
	}

	return nil
}

func fetchLocalImages(ctx context.Context, data *Data, client *client.Client) error {
	images, err := client.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return fmt.Errorf("unable to list Docker images: %w", err)
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

func pullImage(ctx context.Context, data *Data, client *client.Client, authConfig *AuthConfigs, image string, platform string) error {
	pullOpts := parseImageOptions(image)

	auth := types.AuthConfig{}
	if authConfig, ok := authConfig.Configs[pullOpts.Registry]; ok {
		auth = authConfig
	}

	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("error creating auth config: %w", err)
	}

	out, err := client.ImagePull(ctx, image, types.ImagePullOptions{
		RegistryAuth: base64.URLEncoding.EncodeToString(encodedJSON),
		Platform:     platform,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %s: %w", image, err)
	}
	defer out.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(out); err != nil {
		return err
	}
	s := buf.String()
	log.Printf("[DEBUG] pulled image %v: %v", image, s)

	return nil
}

type internalPullImageOptions struct {
	Repository string `qs:"fromImage"`
	Tag        string

	// Only required for Docker Engine 1.9 or 1.10 w/ Remote API < 1.21
	// and Docker Engine < 1.9
	// This parameter was removed in Docker Engine 1.11
	Registry string
}

// Parses an image name into a PullImageOptions struct.
// If the name has no registry, the registry-1.docker.io is used
// If the name has no tag, the tag "latest" is used
func parseImageOptions(image string) internalPullImageOptions {
	pullOpts := internalPullImageOptions{}

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

	if pullOpts.Tag == "" {
		pullOpts.Tag = "latest"
	}

	// Use the official Docker Hub if a registry isn't specified
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry-1.docker.io"
		// Docker prefixes 'library' to official images in the path; 'consul' becomes 'library/consul'
		if !strings.Contains(pullOpts.Repository, "/") {
			pullOpts.Repository = "library/" + pullOpts.Repository
		}
	} else {
		// Otherwise, filter the registry name out of the repo name
		pullOpts.Repository = strings.Replace(pullOpts.Repository, pullOpts.Registry+"/", "", 1)
	}

	return pullOpts
}

func findImage(ctx context.Context, imageName string, client *client.Client, authConfig *AuthConfigs, platform string) (*types.ImageSummary, error) {
	if imageName == "" {
		return nil, fmt.Errorf("empty image name is not allowed")
	}

	var data Data
	// load local images into the data structure
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return nil, err
	}
	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return nil, fmt.Errorf("findImage1: error looking up local image %q: %w", imageName, err)
	}
	if foundImage != nil {
		return foundImage, nil
	}
	if err := pullImage(ctx, &data, client, authConfig, imageName, platform); err != nil {
		return nil, fmt.Errorf("unable to pull image %s: %s", imageName, err)
	}

	// update the data structure of the images
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return nil, err
	}

	foundImage, err = searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return nil, fmt.Errorf("findImage2: error looking up local image %q: %w", imageName, err)
	}
	if foundImage != nil {
		return foundImage, nil
	}

	return nil, fmt.Errorf("unable to find or pull image %s", imageName)
}

func buildDockerImage(ctx context.Context, rawBuild map[string]interface{}, imageName string, client *client.Client) error {
	var (
		err error
	)

	log.Printf("[DEBUG] Building docker image")
	buildOptions := createImageBuildOptions(rawBuild)

	tags := []string{imageName}
	for _, t := range rawBuild["tag"].([]interface{}) {
		tags = append(tags, t.(string))
	}
	buildOptions.Tags = tags

	buildContext := rawBuild["context"].(string)

	enableBuildKitIfSupported(ctx, client, &buildOptions)

	buildCtx, relDockerfile, err := prepareBuildContext(buildContext, buildOptions.Dockerfile)
	if err != nil {
		return err
	}
	buildOptions.Dockerfile = relDockerfile

	var response types.ImageBuildResponse
	response, err = client.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	buildResult, err := decodeBuildMessages(response)
	if err != nil {
		return fmt.Errorf("%s\n\n%s", err, buildResult)
	}
	return nil
}

func enableBuildKitIfSupported(ctx context.Context, client *client.Client, buildOptions *types.ImageBuildOptions) {
	dockerClientVersion := client.ClientVersion()
	log.Printf("[DEBUG] DockerClientVersion: %v, minBuildKitDockerVersion: %v\n", dockerClientVersion, minBuildkitDockerVersion)
	if versions.GreaterThanOrEqualTo(dockerClientVersion, minBuildkitDockerVersion) {
		log.Printf("[DEBUG] Enabling BuildKit")
		s, _ := session.NewSession(ctx, "docker-provider", "")
		dialSession := func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
			return client.DialHijack(ctx, "/session", proto, meta)
		}
		//nolint
		go s.Run(ctx, dialSession)
		defer s.Close()
		buildOptions.SessionID = s.ID()
		buildOptions.Version = types.BuilderBuildKit
	} else {
		buildOptions.Version = types.BuilderV1
	}
}

func prepareBuildContext(specifiedContext string, specifiedDockerfile string) (io.ReadCloser, string, error) {
	var (
		dockerfileCtx io.ReadCloser
		contextDir    string
		relDockerfile string
		err           error
	)
	contextDir, relDockerfile, err = build.GetContextFromLocalDir(specifiedContext, specifiedDockerfile)
	log.Printf("[DEBUG] contextDir %s", contextDir)
	log.Printf("[DEBUG] relDockerfile %s", relDockerfile)
	if err == nil && strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		log.Printf("[DEBUG] Dockerfile is outside of build-context")
		dockerfileCtx, err = os.Open(specifiedDockerfile)
		if err != nil {
			return nil, "", errors.Errorf("unable to open Dockerfile: %v", err)
		}
		defer dockerfileCtx.Close()
	}
	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return nil, "", err
	}

	specifiedDockerfile = archive.CanonicalTarNameForPath(specifiedDockerfile)
	excludes = build.TrimBuildFilesFromExcludes(excludes, specifiedDockerfile, false)
	log.Printf("[DEBUG] Excludes: %v", excludes)
	buildCtx := getBuildContext(contextDir, excludes)

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		log.Printf("[DEBUG] Adding dockerfile to build context")
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return nil, "", err
		}
		return buildCtx, relDockerfile, nil
	}
	return buildCtx, specifiedDockerfile, nil
}

func getBuildContext(filePath string, excludes []string) io.ReadCloser {
	filePath, _ = homedir.Expand(filePath)
	//TarWithOptions works only with absolute paths in Windows.
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Invalid build directory: %s", filePath)
	}
	ctx, _ := archive.TarWithOptions(filePath, &archive.TarOptions{
		ExcludePatterns: excludes,
	})
	return ctx
}

func decodeBuildMessages(response types.ImageBuildResponse) (string, error) {
	buf := new(bytes.Buffer)
	buildErr := error(nil)

	dec := json.NewDecoder(response.Body)
	for dec.More() {
		var m jsonmessage.JSONMessage
		err := dec.Decode(&m)
		if err != nil {
			return buf.String(), fmt.Errorf("problem decoding message from docker daemon: %s", err)
		}

		if err := m.Display(buf, false); err != nil {
			return "", err
		}

		if m.Error != nil {
			buildErr = fmt.Errorf("unable to build image")
		}
	}
	log.Printf("[DEBUG] %s", buf.String())

	return buf.String(), buildErr
}
