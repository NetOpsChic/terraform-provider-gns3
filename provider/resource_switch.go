package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Switch represents the structure for a GNS3 switch node API request/response.
type Switch struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"` // always "ethernet_switch"
	ComputeID string `json:"compute_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
}

// defaultSwitchIcon holds the raw image data for the switch icon.
// Replace "RAW_DATA_FOR_SWITCH_ICON" with your actual raw icon bytes.
var defaultSwitchIcon = []byte("RAW_DATA_FOR_SWITCH_ICON")

// resourceGns3Switch defines the Terraform resource schema for GNS3 switch nodes.
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
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)

	// Build the payload without sending a symbol_id field.
	sw := Switch{
		Name:      name,
		NodeType:  "ethernet_switch",
		ComputeID: computeID,
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

	// Use the fixed symbol_id ":/symbols/classic/ethernet_switch.svg" for the switch.
	symbolID := ":/symbols/classic/ethernet_switch.svg"
	symbolURL := fmt.Sprintf("%s/v2/symbols/%s/raw", host, symbolID)
	req, err := http.NewRequest("POST", symbolURL, bytes.NewBuffer(defaultSwitchIcon))
	if err != nil {
		return fmt.Errorf("failed to create symbol update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{}
	symResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update switch symbol: %s", err)
	}
	defer symResp.Body.Close()
	// Accept either 200 (OK) or 204 (No Content) as success.
	if symResp.StatusCode != http.StatusOK && symResp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := ioutil.ReadAll(symResp.Body)
		return fmt.Errorf("failed to update switch symbol, status code: %d, response: %s", symResp.StatusCode, string(bodyBytes))
	}

	d.SetId(createdSwitch.NodeID)
	d.Set("switch_id", createdSwitch.NodeID)
	return nil
}

func resourceGns3SwitchRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally implement reading to reconcile state.
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
