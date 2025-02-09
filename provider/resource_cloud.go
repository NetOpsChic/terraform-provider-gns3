package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Cloud represents the structure for a GNS3 cloud node API request/response.
type Cloud struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"`  // Specify the node type for a cloud node.
	ComputeID string `json:"compute_id"` // Required field for the API.
	NodeID    string `json:"node_id,omitempty"`
}

// resourceGns3Cloud defines the Terraform resource schema for GNS3 cloud nodes.
func resourceGns3Cloud() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3CloudCreate,
		Read:   resourceGns3CloudRead,
		Delete: resourceGns3CloudDelete,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"compute_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "local",
				ForceNew: true,
			},
			"cloud_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGns3CloudCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	cloudName := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)

	// Build the payload including the node type ("cloud") and compute_id.
	cl := Cloud{
		Name:      cloudName,
		NodeType:  "cloud",
		ComputeID: computeID,
	}

	data, err := json.Marshal(cl)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud node data: %s", err)
	}

	// Use the unified endpoint to create nodes.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating GNS3 cloud node: %s", err)
	}
	defer resp.Body.Close()

	// Expecting a 201 Created response.
	if resp.StatusCode != http.StatusCreated {
		var errorResponse map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		return fmt.Errorf("failed to create cloud node, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	var createdCloud Cloud
	if err := json.NewDecoder(resp.Body).Decode(&createdCloud); err != nil {
		return fmt.Errorf("failed to decode cloud node response: %s", err)
	}

	if createdCloud.NodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	d.SetId(createdCloud.NodeID)
	d.Set("cloud_id", createdCloud.NodeID)
	return nil
}

func resourceGns3CloudRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally implement a read function to reconcile state.
	return nil
}

func resourceGns3CloudDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// Use the unified deletion endpoint.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for cloud node: %s", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete cloud node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete cloud node, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
