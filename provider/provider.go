package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ProviderConfig holds the configuration for the provider
type ProviderConfig struct {
	Host string
}

// Provider function returns the schema.Provider for GNS3
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GNS3_HOST", nil),
				Description: "The GNS3 server host URL.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gns3_project": resourceGns3Project(),
			"gns3_node":    resourceGns3Node(),
			"gns3_link":    resourceGns3Link(),
			"gns3_cloud":   resourceGns3Cloud(),
			"gns3_switch":  resourceGns3Switch(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"gns3_template_id": dataSourceGns3TemplateID(),
		},
		ConfigureFunc: providerConfigure,
	}
}

// providerConfigure initializes the GNS3 client
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &ProviderConfig{
		Host: d.Get("host").(string),
	}
	return config, nil
}
