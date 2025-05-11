package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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

func waitForNode(host, projectID, nodeID string) error {
	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	for i := 0; i < 10; i++ {
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read nodes response: %s", err)
		}
		var nodes []map[string]interface{}
		if err := json.Unmarshal(body, &nodes); err != nil {
			return fmt.Errorf("failed to parse nodes JSON: %s", err)
		}
		for _, node := range nodes {
			if id, ok := node["node_id"].(string); ok && id == nodeID {
				return nil // Node found
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("node %s not found in controller after polling", nodeID)
}

// resourceGns3Link defines the GNS3 link resource schema.
func resourceGns3Link() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3LinkCreate,
		Read:   resourceGns3LinkRead,
		Update: resourceGns3LinkUpdate,
		Delete: resourceGns3LinkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3LinkImporter,
		},

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

	// Retrieve node IDs from resource data
	nodeAID := d.Get("node_a_id").(string)
	nodeBID := d.Get("node_b_id").(string)

	// Poll the controller until both nodes are registered
	if err := waitForNode(host, projectID, nodeAID); err != nil {
		return fmt.Errorf("node A not found: %s", err)
	}
	if err := waitForNode(host, projectID, nodeBID); err != nil {
		return fmt.Errorf("node B not found: %s", err)
	}

	// Build the link payload.
	link := Link{
		Nodes: []LinkNode{
			{
				NodeID:        nodeAID,
				AdapterNumber: d.Get("node_a_adapter").(int),
				PortNumber:    d.Get("node_a_port").(int),
			},
			{
				NodeID:        nodeBID,
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

func resourceGns3LinkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	linkID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/links/%s", host, projectID, linkID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error reading GNS3 link: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Link no longer exists in GNS3, remove from state
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to read link, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Optionally parse and update fields if needed
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

	// Ignore 404 errors during delete — treat as already gone
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete GNS3 link, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	d.SetId("")
	return nil
}
func resourceGns3LinkImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	raw := d.Id()
	var projectID, linkID string

	if parts := strings.SplitN(raw, "/", 2); len(parts) == 2 {
		projectID = parts[0]
		linkID = parts[1]
	} else {
		return nil, fmt.Errorf("invalid import ID format %q: expected <project_id>/<link_id>", raw)
	}

	if err := d.Set("project_id", projectID); err != nil {
		return nil, fmt.Errorf("failed to set project_id during import: %s", err)
	}
	d.SetId(linkID)

	return []*schema.ResourceData{d}, nil
}
