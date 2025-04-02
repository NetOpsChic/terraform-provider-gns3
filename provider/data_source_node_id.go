package provider

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceGns3NodeID fetches a node ID by project and name
func dataSourceGns3NodeID() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGns3NodeIDRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UUID of the project the node belongs to",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the GNS3 node",
			},
			"node_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The resolved node_id of the given node name",
			},
		},
	}
}

func dataSourceGns3NodeIDRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeName := d.Get("name").(string)

	url := fmt.Sprintf("%s/v2/projects/%s/nodes", config.Host, projectID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes from project: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GNS3 API returned non-200 when fetching nodes: %d", resp.StatusCode)
	}

	var nodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return fmt.Errorf("failed to decode response: %s", err)
	}

	for _, node := range nodes {
		if node["name"] == nodeName {
			nodeID := node["node_id"].(string)
			d.SetId(nodeID)
			d.Set("node_id", nodeID)
			return nil
		}
	}

	return fmt.Errorf("node with name '%s' not found in project '%s'", nodeName, projectID)
}
