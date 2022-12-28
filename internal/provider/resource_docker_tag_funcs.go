package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerTagCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	sourceImage := d.Get("source_image").(string)
	targetImage := d.Get("target_image").(string)
	err := client.ImageTag(ctx, sourceImage, targetImage)
	if err != nil {
		return diag.Errorf("failed to create docker tag: %s", err)
	}
	imageInspect, _, err := client.ImageInspectWithRaw(ctx, sourceImage)
	if err != nil {
		return diag.Errorf("failed to ImageInspectWithRaw: %s", err)
	}
	d.Set("source_image_id", imageInspect.ID)
	d.SetId(sourceImage + "." + targetImage)
	return nil
}

func resourceDockerTagDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// We do not delete any of the source and target images
	return nil
}

func resourceDockerTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
