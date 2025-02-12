package provider

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ProviderConfig holds configuration for the provider.
type ProviderConfig struct {
	Host string
}

// Provider returns the Terraform provider for GNS3.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GNS3_HOST", "http://localhost:3080"),
				Description: "The GNS3 server host URL. Default: http://localhost:3080",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gns3_project":   resourceGns3Project(),
			"gns3_cloud":     resourceGns3Cloud(),
			"gns3_switch":    resourceGns3Switch(),
			"gns3_router":    resourceGns3Router(),
			"gns3_link":      resourceGns3Link(),
			"gns3_start_all": resourceGns3StartAll(),
			"gns3_docker":    resourceGns3Docker(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"gns3_template_id": dataSourceGns3TemplateID(),
		},
		ConfigureFunc: providerConfigure,
	}
}

// providerConfigure initializes the provider with the GNS3 host configuration.
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &ProviderConfig{
		Host: d.Get("host").(string),
	}

	log.Printf("[INFO] Terraform GNS3 Provider configured with host: %s", config.Host)
	fmt.Println("[INFO] Terraform GNS3 Provider successfully initialized!")

	return config, nil
}
