package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGns3Node() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3NodeCreate,
		Read:   resourceGns3NodeRead,
		Delete: resourceGns3NodeDelete,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"template_id": {
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
			"x": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ForceNew: true,
			},
			"y": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ForceNew: true,
			},
		},
	}
}

func resourceGns3NodeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig) // Correct type assertion
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Get("template_id").(string)
	nodeName := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int)
	y := d.Get("y").(int)

	// Create node request payload
	nodeData := map[string]interface{}{
		"name":       nodeName,
		"compute_id": computeID,
		"x":          x,
		"y":          y,
	}

	nodeBody, err := json.Marshal(nodeData)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %s", err)
	}

	// Send the request to create the node from the template
	resp, err := http.Post(fmt.Sprintf("%s/v2/projects/%s/templates/%s", host, projectID, templateID), "application/json", bytes.NewBuffer(nodeBody))
	if err != nil {
		return fmt.Errorf("error creating GNS3 node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("failed to create GNS3 node, status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("failed to create GNS3 node, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	// Parse the response to retrieve the node_id
	var createdNode map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdNode); err != nil {
		return fmt.Errorf("error decoding GNS3 API response: %s", err)
	}

	nodeID, exists := createdNode["node_id"].(string)
	if !exists {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	// Set the resource ID in Terraform
	d.SetId(nodeID)
	return nil
}

func resourceGns3NodeRead(d *schema.ResourceData, meta interface{}) error {
	// Implement if needed
	return nil
}

func resourceGns3NodeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig) // Correct type assertion
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID), nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting GNS3 node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete GNS3 node, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
