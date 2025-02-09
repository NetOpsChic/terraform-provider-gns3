package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Switch represents the structure for a GNS3 Ethernet switch API request/response.
type Switch struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"`         // Specify the node type for a switch.
	ComputeID string `json:"compute_id"`        // Required field for the API.
	NodeID    string `json:"node_id,omitempty"` // The returned unique ID for the switch.
}

// resourceGns3Switch defines the Terraform resource schema for GNS3 switches.
func resourceGns3Switch() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3SwitchCreate,
		Read:   resourceGns3SwitchRead,
		Delete: resourceGns3SwitchDelete,
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
			"switch_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGns3SwitchCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	switchName := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)

	// Build the payload including the node type and compute_id.
	sw := Switch{
		Name:      switchName,
		NodeType:  "ethernet_switch",
		ComputeID: computeID,
	}

	data, err := json.Marshal(sw)
	if err != nil {
		return fmt.Errorf("failed to marshal switch data: %s", err)
	}

	// Use the unified endpoint to create nodes.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating GNS3 switch: %s", err)
	}
	defer resp.Body.Close()

	// Expecting a 201 Created response.
	if resp.StatusCode != http.StatusCreated {
		var errorResponse map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		return fmt.Errorf("failed to create switch, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	var createdSwitch Switch
	if err := json.NewDecoder(resp.Body).Decode(&createdSwitch); err != nil {
		return fmt.Errorf("failed to decode switch response: %s", err)
	}

	if createdSwitch.NodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	d.SetId(createdSwitch.NodeID)
	d.Set("switch_id", createdSwitch.NodeID)
	return nil
}

func resourceGns3SwitchRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally implement a read function to reconcile state.
	return nil
}

func resourceGns3SwitchDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// Use the unified deletion endpoint.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for switch: %s", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete switch: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete switch, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
