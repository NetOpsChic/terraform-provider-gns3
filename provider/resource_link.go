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

var linkHTTPClient = &http.Client{Timeout: 15 * time.Second}

func waitForNode(host, projectID, nodeID string) error {
	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	for i := 0; i < 10; i++ {
		resp, err := linkHTTPClient.Get(url)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %s", err)
		}
		body, err := io.ReadAll(resp.Body)
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
				ForceNew:    true,
				Description: "ID of the first node. This can be a router, switch, or cloud node. Changing this destroys and recreates the link — GNS3 accepts PUT requests to rewire an existing link's endpoints but does not actually apply them.",
			},
			"node_a_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Adapter number for the first node. Changing this destroys and recreates the link.",
			},
			"node_a_port": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Port number for the first node. Changing this destroys and recreates the link.",
			},
			"node_b_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the second node. This can be a router, switch, or cloud node. Changing this destroys and recreates the link.",
			},
			"node_b_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Adapter number for the second node. Changing this destroys and recreates the link.",
			},
			"node_b_port": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Port number for the second node. Changing this destroys and recreates the link.",
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

	nodeAID := d.Get("node_a_id").(string)
	nodeBID := d.Get("node_b_id").(string)

	// Poll the controller until both nodes are registered
	if err := waitForNode(host, projectID, nodeAID); err != nil {
		return fmt.Errorf("node A not found: %s", err)
	}
	if err := waitForNode(host, projectID, nodeBID); err != nil {
		return fmt.Errorf("node B not found: %s", err)
	}

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
	resp, err := linkHTTPClient.Post(url, "application/json", bytes.NewBuffer(linkData))
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
	if err := d.Set("link_id", createdLink.LinkID); err != nil {
		return err
	}
	return resourceGns3LinkRead(d, meta)
}

func resourceGns3LinkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	linkID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/links/%s", host, projectID, linkID)
	resp, err := linkHTTPClient.Get(url)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to read link, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var link Link
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return fmt.Errorf("failed to decode link response: %s", err)
	}

	if len(link.Nodes) != 2 {
		return fmt.Errorf("unexpected link response: expected 2 endpoints, got %d", len(link.Nodes))
	}

	// The API doesn't guarantee endpoint ordering matches node_a/node_b as
	// originally specified, so match against whichever endpoint currently
	// corresponds to node_a_id in state to decide which array slot maps to
	// which schema field, rather than assuming index 0 is always "a".
	currentNodeAID := d.Get("node_a_id").(string)
	first, second := link.Nodes[0], link.Nodes[1]
	if first.NodeID != currentNodeAID && second.NodeID == currentNodeAID {
		first, second = second, first
	}

	if err := d.Set("node_a_id", first.NodeID); err != nil {
		return err
	}
	if err := d.Set("node_a_adapter", first.AdapterNumber); err != nil {
		return err
	}
	if err := d.Set("node_a_port", first.PortNumber); err != nil {
		return err
	}
	if err := d.Set("node_b_id", second.NodeID); err != nil {
		return err
	}
	if err := d.Set("node_b_adapter", second.AdapterNumber); err != nil {
		return err
	}
	if err := d.Set("node_b_port", second.PortNumber); err != nil {
		return err
	}
	if err := d.Set("link_id", link.LinkID); err != nil {
		return err
	}

	return nil
}

// resourceGns3LinkUpdate exists only to satisfy the schema.Resource interface.
// Every field on this resource is ForceNew (see schema above) — GNS3 accepts
// PUT requests to rewire an existing link's endpoints but does not actually
// apply them, confirmed via live testing, so Terraform destroys and recreates
// the link instead of calling this function for any real change.
func resourceGns3LinkUpdate(d *schema.ResourceData, meta interface{}) error {
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

	resp, err := linkHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting GNS3 link: %s", err)
	}
	defer resp.Body.Close()

	// Treat "already gone" as success so destroy doesn't fail on drift
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
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
