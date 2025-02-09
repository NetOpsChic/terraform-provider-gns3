package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Cloud represents the structure for a GNS3 cloud node API request/response.
type Cloud struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"` // always "cloud"
	ComputeID string `json:"compute_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
}

// defaultCloudIcon holds the raw image data for the cloud icon.
// Replace the placeholder below with your actual raw icon bytes.
var defaultCloudIcon = []byte("RAW_DATA_FOR_CLOUD_ICON")

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
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)

	// Build the payload without sending symbol_id.
	cl := Cloud{
		Name:      name,
		NodeType:  "cloud",
		ComputeID: computeID,
	}

	data, err := json.Marshal(cl)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud node data: %s", err)
	}

	// Create the cloud node.
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

	// Update the node's symbol using the symbols endpoint.
	// Use the fixed symbol_id ":/symbols/classic/cloud.svg"
	// (URL-encode the symbol if necessary)
	symbolID := ":/symbols/classic/cloud.svg"
	symbolURL := fmt.Sprintf("%s/v2/symbols/%s/raw", host, symbolID)
	req, err := http.NewRequest("POST", symbolURL, bytes.NewBuffer(defaultCloudIcon))
	if err != nil {
		return fmt.Errorf("failed to create symbol update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{}
	symResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update cloud symbol: %s", err)
	}
	defer symResp.Body.Close()
	// Accept either 200 (OK) or 204 (No Content)
	if symResp.StatusCode != http.StatusOK && symResp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := ioutil.ReadAll(symResp.Body)
		return fmt.Errorf("failed to update cloud symbol, status code: %d, response: %s", symResp.StatusCode, string(bodyBytes))
	}

	d.SetId(createdCloud.NodeID)
	d.Set("cloud_id", createdCloud.NodeID)
	return nil
}

func resourceGns3CloudRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally implement reading to reconcile state.
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
