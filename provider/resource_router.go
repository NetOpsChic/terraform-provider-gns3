package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceGns3Router defines the Terraform resource schema for GNS3 routers.
func resourceGns3Router() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3RouterCreate,
		Read:   resourceGns3RouterRead,
		Delete: resourceGns3RouterDelete,

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

func resourceGns3RouterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig) // Correct type assertion
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Get("template_id").(string)
	routerName := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int)
	y := d.Get("y").(int)

	// Create router request payload
	routerData := map[string]interface{}{
		"name":       routerName,
		"compute_id": computeID,
		"x":          x,
		"y":          y,
	}

	nodeBody, err := json.Marshal(routerData)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %s", err)
	}

	// Send the request to create the router from the template
	resp, err := http.Post(fmt.Sprintf("%s/v2/projects/%s/templates/%s", host, projectID, templateID), "application/json", bytes.NewBuffer(nodeBody))
	if err != nil {
		return fmt.Errorf("error creating GNS3 router: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("failed to create GNS3 router, status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("failed to create GNS3 router, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	// Parse the response to retrieve the node_id (router id)
	var createdRouter map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdRouter); err != nil {
		return fmt.Errorf("error decoding GNS3 API response: %s", err)
	}

	routerID, exists := createdRouter["node_id"].(string)
	if !exists || routerID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	// Set the resource ID in Terraform
	d.SetId(routerID)
	return nil
}

func resourceGns3RouterRead(d *schema.ResourceData, meta interface{}) error {
	// Implement if needed to reconcile state.
	return nil
}

func resourceGns3RouterDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig) // Correct type assertion
	host := config.Host
	projectID := d.Get("project_id").(string)
	routerID := d.Id()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, routerID), nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting GNS3 router: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete GNS3 router, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
