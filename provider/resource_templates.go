package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var templateHTTPClient = &http.Client{Timeout: 15 * time.Second}

// resourceGns3Template defines the Terraform resource schema for GNS3 templates.
//
// Note on Update reliability: this resource instantiates a node from a
// GNS3 template, and the resulting node's type (qemu, dynamips, docker,
// nat, etc.) depends entirely on which template was used — it is not fixed
// like the other resources in this provider. We've confirmed elsewhere that
// GNS3 silently ignores rename requests for "nat" type nodes specifically
// (see GNS3/gns3-server#2811) while renames work fine for switch/cloud/qemu.
// If a template instantiates a nat-type node, changing "name" here may hit
// the same silent-no-op behavior. Read is fully synced (see below) so that,
// even if this happens, `terraform plan` will correctly show the persisting
// drift rather than silently believing the rename succeeded.
func resourceGns3Template() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3TemplateCreate,
		Read:   resourceGns3TemplateRead,
		Update: resourceGns3TemplateUpdate,
		Delete: resourceGns3TemplateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3TemplateImporter,
		},

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
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the node created from the template. Note: if the template instantiates a NAT-type node, GNS3 may silently ignore rename requests — see GNS3/gns3-server#2811.",
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

	url := fmt.Sprintf("%s/v2/projects/%s/templates/%s", host, projectID, templateID)
	resp, err := templateHTTPClient.Post(url, "application/json", bytes.NewBuffer(nodeBody))
	if err != nil {
		return fmt.Errorf("error creating GNS3 template: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create GNS3 template, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var createdTemplate map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdTemplate); err != nil {
		return fmt.Errorf("error decoding GNS3 API response: %s", err)
	}
	templateNodeID, exists := createdTemplate["node_id"].(string)
	if !exists || templateNodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	d.SetId(templateNodeID)

	if d.Get("start").(bool) {
		startURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/start", host, projectID, templateNodeID)
		startResp, err := templateHTTPClient.Post(startURL, "application/json", nil)
		if err != nil {
			return fmt.Errorf("error starting node: %s", err)
		}
		defer startResp.Body.Close()
		if startResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(startResp.Body)
			return fmt.Errorf("failed to start node, status code: %d, response: %s", startResp.StatusCode, string(body))
		}
	}

	return resourceGns3TemplateRead(d, meta)
}

func resourceGns3TemplateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	resp, err := templateHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("error reading GNS3 node (template): %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to read template node, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode template node response: %s", err)
	}

	if v, ok := node["name"].(string); ok {
		if err := d.Set("name", v); err != nil {
			return err
		}
	}
	if v, ok := node["compute_id"].(string); ok {
		if err := d.Set("compute_id", v); err != nil {
			return err
		}
	}
	if v, ok := node["x"].(float64); ok {
		if err := d.Set("x", int(v)); err != nil {
			return err
		}
	}
	if v, ok := node["y"].(float64); ok {
		if err := d.Set("y", int(v)); err != nil {
			return err
		}
	}
	if v, ok := node["status"].(string); ok {
		if err := d.Set("start", v == "started"); err != nil {
			return err
		}
	}

	return nil
}

func resourceGns3TemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	templateID := d.Id()

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

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, templateID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := templateHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update template: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update template, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Handle "start" toggle separately — it's a lifecycle action, not a
	// property the generic node PUT endpoint controls.
	if d.HasChange("start") {
		action := "stop"
		if d.Get("start").(bool) {
			action = "start"
		}
		actionURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/%s", host, projectID, templateID, action)
		actionReq, err := http.NewRequest("POST", actionURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create %s request: %s", action, err)
		}
		actionResp, err := templateHTTPClient.Do(actionReq)
		if err != nil {
			return fmt.Errorf("failed to %s node: %s", action, err)
		}
		defer actionResp.Body.Close()
		if actionResp.StatusCode != http.StatusOK && actionResp.StatusCode != http.StatusConflict {
			body, _ := io.ReadAll(actionResp.Body)
			return fmt.Errorf("failed to %s node, status code: %d, response: %s", action, actionResp.StatusCode, string(body))
		}
	}

	// Re-read to sync state — this is what would surface a silent-no-op
	// rename (e.g. if the template instantiated a nat-type node) as
	// persisting drift, rather than the previous stub Read hiding it.
	return resourceGns3TemplateRead(d, meta)
}

func resourceGns3TemplateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for template node: %s", err)
	}

	resp, err := templateHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete template node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete template node, status code: %d, body: %s", resp.StatusCode, body)
	}

	d.SetId("")
	return nil
}

func resourceGns3TemplateImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	raw := d.Id()
	var projectID, nodeID string

	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		projectID = parts[0]
		nodeID = parts[1]
	} else {
		return nil, fmt.Errorf("invalid ID format %q — expected <project_id>/<node_id>", raw)
	}

	if err := d.Set("project_id", projectID); err != nil {
		return nil, err
	}
	d.SetId(nodeID)
	return []*schema.ResourceData{d}, nil
}
