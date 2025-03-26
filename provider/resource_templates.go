package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceGns3Template defines the Terraform resource schema for GNS3 templates.
func resourceGns3Template() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3TemplateCreate,
		Read:   resourceGns3TemplateRead,
		Update: resourceGns3TemplateUpdate,
		Delete: resourceGns3TemplateDelete,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"template_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, // Ensures deletion & recreation if template_id changes
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"compute_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "local",
			},
			"start": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"x": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"y": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

func resourceGns3TemplateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Get("template_id").(string)
	templateName := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int)
	y := d.Get("y").(int)

	// Create template request payload
	templateData := map[string]interface{}{
		"name":       templateName,
		"compute_id": computeID,
		"x":          x,
		"y":          y,
	}

	nodeBody, err := json.Marshal(templateData)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %s", err)
	}

	// Send the request to create the template
	resp, err := http.Post(fmt.Sprintf("%s/v2/projects/%s/templates/%s", host, projectID, templateID), "application/json", bytes.NewBuffer(nodeBody))
	if err != nil {
		return fmt.Errorf("error creating GNS3 template: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create GNS3 template, status code: %d", resp.StatusCode)
	}

	// Parse the response to retrieve the node_id (template ID)
	var createdTemplate map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdTemplate); err != nil {
		return fmt.Errorf("error decoding GNS3 API response: %s", err)
	}
	templateNodeID, exists := createdTemplate["node_id"].(string)
	if !exists || templateNodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	// Set the resource ID in Terraform
	d.SetId(templateNodeID)

	// Check if the "start" attribute is true and start the node if so.
	if d.Get("start").(bool) {
		startURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/start", host, projectID, templateNodeID)
		startResp, err := http.Post(startURL, "application/json", nil)
		if err != nil {
			return fmt.Errorf("error starting node: %s", err)
		}
		defer startResp.Body.Close()
		if startResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to start node, status code: %d", startResp.StatusCode)
		}
	}

	return nil
}

func resourceGns3TemplateRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally implement a call to refresh the template state from the API.
	return nil
}

func resourceGns3TemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Id()

	// Build the update payload with the updated attributes.
	updateData := map[string]interface{}{
		"name":       d.Get("name").(string),
		"compute_id": d.Get("compute_id").(string),
		"x":          d.Get("x").(int),
		"y":          d.Get("y").(int),
	}

	data, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %s", err)
	}

	// Send a PUT request to update the template.
	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, templateID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update template: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update template, status code: %d", resp.StatusCode)
	}

	// Optionally, re-read the resource to update state.
	return resourceGns3TemplateRead(d, meta)
}

func resourceGns3TemplateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Id()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, templateID), nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting GNS3 template: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete GNS3 template, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
