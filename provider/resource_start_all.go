package provider

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceGns3StartAll defines a resource that starts all nodes in a project.
func resourceGns3StartAll() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3StartAllCreate,
		Read:   resourceGns3StartAllRead,
		Delete: resourceGns3StartAllDelete,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the GNS3 project whose nodes should be started.",
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
	// Optionally, you could implement a check to verify that nodes are started.
	// For now, this is a no-op.
	return nil
}

func resourceGns3StartAllDelete(d *schema.ResourceData, meta interface{}) error {
	// Optionally, implement a stop action here (e.g., POST to /nodes/stop)
	// For now, we simply remove the resource.
	d.SetId("")
	return nil
}
