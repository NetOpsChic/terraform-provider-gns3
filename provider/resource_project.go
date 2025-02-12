package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Project represents the structure for GNS3 project API requests/responses.
type Project struct {
	Name      string `json:"name"`
	ProjectID string `json:"project_id,omitempty"`
}

// resourceGns3Project defines the Terraform resource schema for GNS3 projects.
func resourceGns3Project() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3ProjectCreate,
		Read:   resourceGns3ProjectRead,
		Update: resourceGns3ProjectUpdate,
		Delete: resourceGns3ProjectDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the GNS3 project.",
				// ForceNew removed so that the project name can be updated.
			},
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID assigned by GNS3 to the project.",
			},
		},
	}
}

// resourceGns3ProjectCreate creates a new GNS3 project.
func resourceGns3ProjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectName := d.Get("name").(string)

	// Prepare project request payload.
	project := Project{
		Name: projectName,
	}

	projectData, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects", host)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(projectData))
	if err != nil {
		return fmt.Errorf("failed to create project: %s", err)
	}
	defer resp.Body.Close()

	var createdProject map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdProject); err != nil {
		return err
	}

	// Retrieve the project ID using either "project_id" or "projectId".
	projectID, exists := createdProject["project_id"].(string)
	if !exists || projectID == "" {
		projectID, exists = createdProject["projectId"].(string)
		if !exists || projectID == "" {
			return fmt.Errorf("failed to retrieve project_id from GNS3 API response: %v", createdProject)
		}
	}

	d.SetId(projectID)
	d.Set("project_id", projectID)
	return nil
}

// resourceGns3ProjectRead reads the project state from GNS3.
// (This implementation can be enhanced to refresh state if needed.)
// resourceGns3ProjectRead reads the project state from GNS3.
func resourceGns3ProjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

	// If there is no ID, assume the project doesn't exist
	if projectID == "" {
		return nil
	}

	// Fetch project details from GNS3
	url := fmt.Sprintf("%s/v2/projects/%s", host, projectID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to read project from GNS3: %s", err)
	}
	defer resp.Body.Close()

	// If project does not exist, remove it from Terraform state
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to retrieve project, status code: %d", resp.StatusCode)
	}

	// Decode response
	var project map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return fmt.Errorf("failed to decode project response: %s", err)
	}

	// Ensure project still exists and update state
	if project["project_id"] == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", project["name"])
	d.Set("project_id", project["project_id"])

	return nil
}

// resourceGns3ProjectUpdate updates the project's name in place.
func resourceGns3ProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

	// Check if the "name" attribute has changed.
	if d.HasChange("name") {
		newName := d.Get("name").(string)
		updateData := map[string]interface{}{
			"name": newName,
		}
		data, err := json.Marshal(updateData)
		if err != nil {
			return fmt.Errorf("failed to marshal update data: %s", err)
		}

		url := fmt.Sprintf("%s/v2/projects/%s", host, projectID)
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to create update request: %s", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to update project: %s", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResponse map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
			return fmt.Errorf("failed to update project, status code: %d, error: %v", resp.StatusCode, errorResponse)
		}
	}

	return resourceGns3ProjectRead(d, meta)
}

// resourceGns3ProjectDelete deletes the project from GNS3.
func resourceGns3ProjectDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s", host, projectID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %s", err)
	}
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
