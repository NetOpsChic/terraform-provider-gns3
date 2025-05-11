package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Switch represents a GNS3 switch node API request/response.
type Switch struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"`
	ComputeID string `json:"compute_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	X         int    `json:"x,omitempty"` // ✅ Added X coordinate
	Y         int    `json:"y,omitempty"` // ✅ Added Y coordinate
}

// Default switch symbol icon
var defaultSwitchIcon = []byte("RAW_DATA_FOR_SWITCH_ICON")

// resourceGns3Switch defines the Terraform resource schema for GNS3 switch nodes.
func resourceGns3Switch() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3SwitchCreate,
		Read:   resourceGns3SwitchRead,
		Update: resourceGns3SwitchUpdate,
		Delete: resourceGns3SwitchDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project ID where the switch is deployed.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the switch node.",
			},
			"compute_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "local",
				Description: "Compute ID where the switch node is running.",
			},
			"x": { // ✅ Added X coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "X position of the switch node in GNS3 GUI.",
			},
			"y": { // ✅ Added Y coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Y position of the switch node in GNS3 GUI.",
			},
			"switch_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The switch node's ID assigned by GNS3.",
			},
		},
	}
}

func resourceGns3SwitchCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int) // ✅ Retrieve X coordinate
	y := d.Get("y").(int) // ✅ Retrieve Y coordinate

	// Build the payload with X and Y coordinates
	sw := Switch{
		Name:      name,
		NodeType:  "ethernet_switch",
		ComputeID: computeID,
		X:         x,
		Y:         y,
	}

	data, err := json.Marshal(sw)
	if err != nil {
		return fmt.Errorf("failed to marshal switch data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating GNS3 switch: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create switch, status code: %d, error: %v", resp.StatusCode, errResp)
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

// Update function for modifying existing switch nodes
func resourceGns3SwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	switchID := d.Id()

	updateData := map[string]interface{}{}

	if d.HasChange("name") {
		updateData["name"] = d.Get("name").(string)
	}

	if d.HasChange("compute_id") {
		updateData["compute_id"] = d.Get("compute_id").(string)
	}

	if d.HasChange("x") {
		updateData["x"] = d.Get("x").(int) // ✅ Update X coordinate
	}

	if d.HasChange("y") {
		updateData["y"] = d.Get("y").(int) // ✅ Update Y coordinate
	}

	if len(updateData) == 0 {
		return nil
	}

	updateBody, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, switchID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(updateBody))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error updating GNS3 switch node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to update switch node, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resourceGns3SwitchRead(d, meta)
}

func resourceGns3SwitchRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to read switch node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Node no longer exists
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to read switch node, status code: %d, body: %s", resp.StatusCode, body)
	}

	// Optional: parse attributes and update d.Set(...)
	return nil
}

func resourceGns3SwitchDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

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
