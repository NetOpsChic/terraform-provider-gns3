package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceGns3LinkID defines a data source for retrieving a GNS3 link's ID.
// It queries the controller API endpoint to get all links for a given project and then
// searches for a link matching the given "name" (you may adjust the matching criteria as needed).
func dataSourceGns3LinkID() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGns3LinkIDRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UUID of the GNS3 project in which to search for the link.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the link to search for.",
			},
			"link_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the link returned by the GNS3 API.",
			},
		},
	}
}

func dataSourceGns3LinkIDRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	linkName := d.Get("name").(string)

	// Construct the API URL using the controller endpoint.
	apiURL := fmt.Sprintf("%s/v2/controller/link/projects/%s/links", config.APIURL, projectID)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to query links: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to query links, status: %d, response: %s", resp.StatusCode, body)
	}

	// Decode the JSON response into a slice of link objects.
	var links []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return fmt.Errorf("failed to decode links: %s", err)
	}

	// Loop through the links to find one that matches the given name.
	// (Adjust this matching logic as needed—for example, you might check endpoints if there is no name.)
	for _, link := range links {
		if name, ok := link["name"].(string); ok && name == linkName {
			if id, ok := link["link_id"].(string); ok && id != "" {
				d.SetId(id)
				d.Set("link_id", id)
				return nil
			}
		}
	}

	return fmt.Errorf("link with name '%s' not found in project '%s'", linkName, projectID)
}
