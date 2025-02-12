package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// LinkNode represents a node in a GNS3 link.
type LinkNode struct {
	NodeID        string `json:"node_id"`
	AdapterNumber int    `json:"adapter_number"`
	PortNumber    int    `json:"port_number"`
}

// Link represents a GNS3 link between nodes.
type Link struct {
	LinkID string     `json:"link_id,omitempty"`
	Nodes  []LinkNode `json:"nodes"`
}

// resourceGns3Link defines the GNS3 link resource schema.
func resourceGns3Link() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3LinkCreate,
		Read:   resourceGns3LinkRead,
		Update: resourceGns3LinkUpdate,
		Delete: resourceGns3LinkDelete,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project ID in which the link is created.",
			},
			"node_a_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the first node. This can be a router, switch, or cloud node.",
			},
			"node_a_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Adapter number for the first node.",
			},
			"node_a_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Port number for the first node.",
			},
			"node_b_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the second node. This can be a router, switch, or cloud node.",
			},
			"node_b_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Adapter number for the second node.",
			},
			"node_b_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Port number for the second node.",
			},
			"link_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique ID of the link returned by the GNS3 API.",
			},
		},
	}
}

// resourceGns3LinkCreate creates a new link between two nodes.
func resourceGns3LinkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)

	// Build the link payload.
	link := Link{
		Nodes: []LinkNode{
			{
				NodeID:        d.Get("node_a_id").(string),
				AdapterNumber: d.Get("node_a_adapter").(int),
				PortNumber:    d.Get("node_a_port").(int),
			},
			{
				NodeID:        d.Get("node_b_id").(string),
				AdapterNumber: d.Get("node_b_adapter").(int),
				PortNumber:    d.Get("node_b_port").(int),
			},
		},
	}

	linkData, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("failed to marshal link data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/links", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(linkData))
	if err != nil {
		return fmt.Errorf("failed to create link: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("failed to create link, status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("failed to create link, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	var createdLink Link
	if err := json.NewDecoder(resp.Body).Decode(&createdLink); err != nil {
		return fmt.Errorf("failed to decode link response: %s", err)
	}

	d.SetId(createdLink.LinkID)
	d.Set("link_id", createdLink.LinkID)
	return nil
}

// resourceGns3LinkRead reads the link resource. This implementation is currently a no-op.
func resourceGns3LinkRead(d *schema.ResourceData, meta interface{}) error {
	// Optionally, implement a GET request to refresh state.
	return nil
}

// resourceGns3LinkUpdate updates an existing link with new parameters.
func resourceGns3LinkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	linkID := d.Id()

	// Build the update payload with the updated attributes.
	link := Link{
		Nodes: []LinkNode{
			{
				NodeID:        d.Get("node_a_id").(string),
				AdapterNumber: d.Get("node_a_adapter").(int),
				PortNumber:    d.Get("node_a_port").(int),
			},
			{
				NodeID:        d.Get("node_b_id").(string),
				AdapterNumber: d.Get("node_b_adapter").(int),
				PortNumber:    d.Get("node_b_port").(int),
			},
		},
	}

	linkData, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("failed to marshal update link data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/links/%s", host, projectID, linkID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(linkData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update link: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		return fmt.Errorf("failed to update link, status code: %d, error: %v", resp.StatusCode, errorResponse)
	}

	// Optionally re-read the resource state.
	return resourceGns3LinkRead(d, meta)
}

// resourceGns3LinkDelete deletes the link.
func resourceGns3LinkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	linkID := d.Id()

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s/links/%s", host, projectID, linkID), nil)
	if err != nil {
		return fmt.Errorf("error creating delete request: %s", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting GNS3 link: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete GNS3 link, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}
