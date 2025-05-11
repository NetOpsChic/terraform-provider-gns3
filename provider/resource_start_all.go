package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceGns3StartAll defines a resource that starts all nodes in a project.
func resourceGns3StartAll() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3StartAllCreate,
		Read:   resourceGns3StartAllRead,
		Update: resourceGns3StartAllUpdate,
		Delete: resourceGns3StartAllDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3StartAllImporter,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the GNS3 project whose nodes should be started.",
				// Removed ForceNew to allow updates (e.g. re-trigger start if project_id changes).
			},
		},
	}
}

func resourceGns3StartAllCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)

	// Build the URL for starting all nodes.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes/start", host, projectID)

	// The API may expect an empty JSON object; adjust as needed.
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to start all nodes: %s", err)
	}
	defer resp.Body.Close()

	// Accept either 200 OK or 204 No Content as success.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to start all nodes, status code: %d", resp.StatusCode)
	}

	// Use a computed ID based on the project ID.
	d.SetId(projectID + "-start")
	return nil
}

func resourceGns3StartAllRead(d *schema.ResourceData, meta interface{}) error {
	// This is an action resource; optionally implement a check to verify nodes are started.
	return nil
}

func resourceGns3StartAllUpdate(d *schema.ResourceData, meta interface{}) error {
	// For updates, we re-trigger the start action.
	return resourceGns3StartAllCreate(d, meta)
}

func resourceGns3StartAllDelete(d *schema.ResourceData, meta interface{}) error {
	// Optionally, implement a "stop" action if supported.
	// For now, we'll simply remove the resource from state.
	d.SetId("")
	return nil
}
func resourceGns3StartAllImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	projectID := d.Id()

	if projectID == "" {
		return nil, fmt.Errorf("missing project_id for gns3_start_all import")
	}

	if err := d.Set("project_id", projectID); err != nil {
		return nil, fmt.Errorf("failed to set project_id: %s", err)
	}

	d.SetId(projectID + "-start")
	return []*schema.ResourceData{d}, nil
}
