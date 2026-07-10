package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceGns3LinkID looks up an existing GNS3 link by its two endpoints
// (node/adapter/port pairs). GNS3 links have no "name" field, so identity is
// necessarily defined by what they connect, not by a label — this matches
// the same node_a_id/node_a_adapter/node_a_port/node_b_id/... fields used to
// create a link via the gns3_link resource.
func dataSourceGns3LinkID() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGns3LinkIDRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UUID of the GNS3 project in which to search for the link.",
			},
			"node_a_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Node ID of one endpoint of the link.",
			},
			"node_a_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Adapter number on node_a.",
			},
			"node_a_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Port number on node_a.",
			},
			"node_b_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Node ID of the other endpoint of the link.",
			},
			"node_b_adapter": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Adapter number on node_b.",
			},
			"node_b_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Port number on node_b.",
			},
			"link_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the matching link, as returned by the GNS3 API.",
			},
		},
	}
}

// linkEndpoint mirrors one element of a link's "nodes" array in the GNS3 API response.
type linkEndpoint struct {
	NodeID        string `json:"node_id"`
	AdapterNumber int    `json:"adapter_number"`
	PortNumber    int    `json:"port_number"`
}

func (e linkEndpoint) matches(nodeID string, adapter, port int) bool {
	return e.NodeID == nodeID && e.AdapterNumber == adapter && e.PortNumber == port
}

func dataSourceGns3LinkIDRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)

	wantA := linkEndpoint{
		NodeID:        d.Get("node_a_id").(string),
		AdapterNumber: d.Get("node_a_adapter").(int),
		PortNumber:    d.Get("node_a_port").(int),
	}
	wantB := linkEndpoint{
		NodeID:        d.Get("node_b_id").(string),
		AdapterNumber: d.Get("node_b_adapter").(int),
		PortNumber:    d.Get("node_b_port").(int),
	}

	// Correct controller endpoint for listing links in a project. (The
	// previous implementation used a fabricated "/v2/controller/link/..."
	// path that does not exist in the GNS3 API and would 404 on every call.)
	apiURL := fmt.Sprintf("%s/v2/projects/%s/links", config.Host, projectID)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to query links: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to query links, status: %d, response: %s", resp.StatusCode, body)
	}

	var links []struct {
		LinkID string         `json:"link_id"`
		Nodes  []linkEndpoint `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return fmt.Errorf("failed to decode links: %s", err)
	}

	for _, link := range links {
		if len(link.Nodes) != 2 {
			continue
		}
		n0, n1 := link.Nodes[0], link.Nodes[1]

		// Endpoint order in the API response isn't guaranteed to match the
		// order the caller specified node_a/node_b in, so check both orderings.
		matchesForward := n0.matches(wantA.NodeID, wantA.AdapterNumber, wantA.PortNumber) &&
			n1.matches(wantB.NodeID, wantB.AdapterNumber, wantB.PortNumber)
		matchesReverse := n0.matches(wantB.NodeID, wantB.AdapterNumber, wantB.PortNumber) &&
			n1.matches(wantA.NodeID, wantA.AdapterNumber, wantA.PortNumber)

		if matchesForward || matchesReverse {
			if link.LinkID == "" {
				return fmt.Errorf("matching link found but has an empty link_id")
			}
			d.SetId(link.LinkID)
			if err := d.Set("link_id", link.LinkID); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf(
		"no link found in project '%s' connecting node %s (adapter %d, port %d) to node %s (adapter %d, port %d)",
		projectID, wantA.NodeID, wantA.AdapterNumber, wantA.PortNumber, wantB.NodeID, wantB.AdapterNumber, wantB.PortNumber,
	)
}
