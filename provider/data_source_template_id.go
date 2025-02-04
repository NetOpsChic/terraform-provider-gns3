package provider

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceGns3TemplateID defines the GNS3 template data source
func dataSourceGns3TemplateID() *schema.Resource {
    return &schema.Resource{
        Read: dataSourceGns3TemplateIDRead,
        Schema: map[string]*schema.Schema{
            "name": {
                Type:     schema.TypeString,
                Required: true,
            },
            "template_id": {
                Type:     schema.TypeString,
                Computed: true,
            },
        },
    }
}

func dataSourceGns3TemplateIDRead(d *schema.ResourceData, meta interface{}) error {
    config := meta.(*ProviderConfig) // Assert meta to *ProviderConfig
    templateName := d.Get("name").(string)

    // Fetch the list of templates from the GNS3 server
    resp, err := http.Get(fmt.Sprintf("%s/v2/templates", config.Host))
    if err != nil {
        return fmt.Errorf("error fetching templates from GNS3 server: %s", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-200 response from GNS3 server: %d %s", resp.StatusCode, resp.Status)
    }

    var templates []map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
        return fmt.Errorf("error decoding response from GNS3 server: %s", err)
    }

    // Search for the template by name
    for _, template := range templates {
        if template["name"] == templateName {
            templateID, ok := template["template_id"].(string)
            if !ok {
                return fmt.Errorf("template_id is not a string for template '%s'", templateName)
            }
            d.SetId(templateID)
            d.Set("template_id", templateID)
            return nil
        }
    }

    return fmt.Errorf("template with name '%s' not found", templateName)
}
