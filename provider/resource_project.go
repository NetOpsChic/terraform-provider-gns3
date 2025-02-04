package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Project structure for API requests
type Project struct {
	Name      string `json:"name"`
	ProjectID string `json:"project_id,omitempty"`
}

// Define the Terraform resource schema for GNS3 projects
func resourceGns3Project() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3ProjectCreate,
		Read:   resourceGns3ProjectRead,
		Delete: resourceGns3ProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGns3ProjectCreate(d *schema.ResourceData, meta interface{}) error {
	// Correct type assertion: meta is *ProviderConfig
	config := meta.(*ProviderConfig)
	host := config.Host
	projectName := d.Get("name").(string)

	// Prepare project request payload
	project := Project{
		Name: projectName,
	}

	projectData, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project data: %s", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/v2/projects", host), "application/json", bytes.NewBuffer(projectData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var createdProject map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdProject); err != nil {
		return err
	}

	// Ensure project_id is properly retrieved
	projectID, exists := createdProject["project_id"].(string)
	if !exists {
		return fmt.Errorf("failed to retrieve project_id from GNS3 API response")
	}

	d.SetId(projectID)
	d.Set("project_id", projectID)
	return nil
}

// Read function for GNS3 project
func resourceGns3ProjectRead(d *schema.ResourceData, meta interface{}) error {
	// Implement if needed
	return nil
}

// Delete function for GNS3 project
func resourceGns3ProjectDelete(d *schema.ResourceData, meta interface{}) error {
	// Correct type assertion: meta is *ProviderConfig, not a string
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s", host, projectID), nil)
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
