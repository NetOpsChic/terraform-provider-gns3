package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			},
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID assigned by GNS3 to the project.",
			},
		},
	}
}

func resourceGns3ProjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectName := d.Get("name").(string)

	// Step 1: Create on controller
	project := Project{Name: projectName}
	projectData, _ := json.Marshal(project)
	controllerResp, _ := http.Post(fmt.Sprintf("%s/v2/projects", host), "application/json", bytes.NewBuffer(projectData))
	defer controllerResp.Body.Close()

	if controllerResp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(controllerResp.Body)
		return fmt.Errorf("Controller project create failed: %s", body)
	}

	var projectResp map[string]interface{}
	json.NewDecoder(controllerResp.Body).Decode(&projectResp)
	projectID := projectResp["project_id"].(string)
	d.SetId(projectID)
	d.Set("project_id", projectID)

	// Step 2: Create on compute
	computePayload := Project{Name: projectName, ProjectID: projectID}
	computeData, _ := json.Marshal(computePayload)
	computeResp, _ := http.Post(fmt.Sprintf("%s/v2/compute/projects", host), "application/json", bytes.NewBuffer(computeData))
	defer computeResp.Body.Close()

	if computeResp.StatusCode != http.StatusCreated && computeResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(computeResp.Body)
		return fmt.Errorf("Compute project create failed: %s", body)
	}

	// Step 3: Open the project (optional but useful)
	// Sync with controller by re-opening the project
	openURL := fmt.Sprintf("%s/v2/projects/%s/open", config.Host, projectID)
	openReq, err := http.NewRequest("POST", openURL, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare open project request to controller: %s", err)
	}

	openResp, err := http.DefaultClient.Do(openReq)
	if err != nil {
		return fmt.Errorf("failed to open/sync project with controller: %s", err)
	}
	defer openResp.Body.Close()

	if openResp.StatusCode != http.StatusOK && openResp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(openResp.Body)
		return fmt.Errorf("failed to open/sync project on controller, status: %d, response: %s", openResp.StatusCode, string(body))
	}

	return nil
}

// resourceGns3ProjectRead reads the project state from GNS3.
func resourceGns3ProjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

	if projectID == "" {
		return nil
	}

	url := fmt.Sprintf("%s/v2/projects/%s", host, projectID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to read project from GNS3: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to retrieve project, status code: %d", resp.StatusCode)
	}

	var project map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return fmt.Errorf("failed to decode project response: %s", err)
	}

	if project["project_id"] == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", project["name"])
	d.Set("project_id", project["project_id"])

	return nil
}

// resourceGns3ProjectUpdate updates the project's name.
func resourceGns3ProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Id()

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
