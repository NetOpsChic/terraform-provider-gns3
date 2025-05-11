package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Cloud represents a GNS3 cloud node API request/response.
type Cloud struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"`
	ComputeID string `json:"compute_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	X         int    `json:"x,omitempty"` // ✅ Added X coordinate
	Y         int    `json:"y,omitempty"` // ✅ Added Y coordinate
}

// Default cloud symbol icon
var defaultCloudIcon = []byte("RAW_DATA_FOR_CLOUD_ICON")

// resourceGns3Cloud defines the Terraform resource schema for GNS3 cloud nodes.
func resourceGns3Cloud() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3CloudCreate,
		Read:   resourceGns3CloudRead,
		Update: resourceGns3CloudUpdate,
		Delete: resourceGns3CloudDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project ID where the cloud node is deployed.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the cloud node.",
			},
			"compute_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "local",
				Description: "Compute ID where the cloud node is running.",
			},
			"x": { // ✅ Added X coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "X position of the cloud node in GNS3 GUI.",
			},
			"y": { // ✅ Added Y coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Y position of the cloud node in GNS3 GUI.",
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cloud node's ID assigned by GNS3.",
			},
		},
	}
}

func resourceGns3CloudCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int) // ✅ Retrieve X coordinate
	y := d.Get("y").(int) // ✅ Retrieve Y coordinate

	cloud := Cloud{
		Name:      name,
		NodeType:  "cloud",
		ComputeID: computeID,
		X:         x, // ✅ Add X coordinate to request
		Y:         y, // ✅ Add Y coordinate to request
	}

	data, err := json.Marshal(cloud)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud node data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating GNS3 cloud node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create cloud node, status code: %d, error: %v", resp.StatusCode, errResp)
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

// Update function for modifying existing cloud nodes
func resourceGns3CloudUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	cloudID := d.Id()

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

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, cloudID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(updateBody))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error updating GNS3 cloud node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to update cloud node, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resourceGns3CloudRead(d, meta)
}

func resourceGns3CloudRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error reading cloud node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Node no longer exists in GNS3 — mark resource as gone
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected read status %d: %s", resp.StatusCode, body)
	}

	return nil
}

func resourceGns3CloudDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

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
