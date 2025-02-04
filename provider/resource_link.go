package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// LinkNode represents a node in a GNS3 link
type LinkNode struct {
	NodeID        string `json:"node_id"`
	AdapterNumber int    `json:"adapter_number"`
	PortNumber    int    `json:"port_number"`
}

// Link represents a GNS3 link between nodes
type Link struct {
	LinkID string     `json:"link_id,omitempty"`
	Nodes  []LinkNode `json:"nodes"`
}

// Define the GNS3 link resource schema
func resourceGns3Link() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3LinkCreate,
		Read:   resourceGns3LinkRead,
		Delete: resourceGns3LinkDelete,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_a_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_a_adapter": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"node_a_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"node_b_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_b_adapter": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"node_b_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"link_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// Create a new link between two nodes
func resourceGns3LinkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)

	// Access the Host from the config
	host := config.Host
	projectID := d.Get("project_id").(string)

	// Define the link structure with adapter and port numbers
	link := Link{
		Nodes: []LinkNode{
			{
				AdapterNumber: d.Get("node_a_adapter").(int),
				NodeID:        d.Get("node_a_id").(string),
				PortNumber:    d.Get("node_a_port").(int),
			},
			{
				AdapterNumber: d.Get("node_b_adapter").(int),
				NodeID:        d.Get("node_b_id").(string),
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
		return fmt.Errorf("failed to decode response: %s", err)
	}

	d.SetId(createdLink.LinkID)
	d.Set("link_id", createdLink.LinkID)
	return nil
}

// Read function for GNS3 link (optional)
func resourceGns3LinkRead(d *schema.ResourceData, meta interface{}) error {
	// Implement if needed
	return nil
}

// Delete function for GNS3 link
func resourceGns3LinkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	linkID := d.Id()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/projects/%s/links/%s", host, projectID, linkID), nil)
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
