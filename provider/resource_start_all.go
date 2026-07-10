package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var startAllHTTPClient = &http.Client{Timeout: 15 * time.Second}

// resourceGns3StartAll defines a resource that starts all nodes in a project.
//
// This is an action resource, not a state resource — it models a one-time
// side effect ("start everything in this project") rather than an object
// GNS3 owns that can be read back and diffed. Two consequences worth knowing:
//
//  1. No drift detection: if nodes are later stopped (manually, or via
//     another resource's start/start_vm flag going false), this resource
//     will not notice and will not re-trigger. Read is intentionally a no-op
//     because there's no meaningful "did I start the nodes" state to query —
//     Update only re-fires if project_id itself changes, which in practice
//     will rarely if ever happen.
//  2. Delete does not stop anything: destroying this resource only removes
//     it from Terraform state. It does not stop or otherwise affect the
//     nodes it started. `terraform destroy` will not shut down your lab.
func resourceGns3StartAll() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3StartAllCreate,
		Read:   resourceGns3StartAllRead,
		Update: resourceGns3StartAllUpdate,
		Delete: resourceGns3StartAllDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3StartAllImporter,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the GNS3 project whose nodes should be started. Note: this resource has no drift detection and does not stop nodes on destroy — see resource-level documentation.",
			},
		},
	}
}

func resourceGns3StartAllCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/start", host, projectID)

	resp, err := startAllHTTPClient.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to start all nodes: %s", err)
	}
	defer resp.Body.Close()

	// Confirmed against GNS3's documented API: this endpoint returns 204 on
	// success. 200 accepted too, defensively, in case behavior differs
	// across server versions.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start all nodes, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	d.SetId(projectID + "-start")
	return nil
}

func resourceGns3StartAllRead(d *schema.ResourceData, meta interface{}) error {
	// Intentional no-op — see the doc comment on resourceGns3StartAll for why.
	return nil
}

func resourceGns3StartAllUpdate(d *schema.ResourceData, meta interface{}) error {
	// Only reachable if project_id changes, since it's the only schema field.
	// Re-trigger the start action against the (new) project.
	return resourceGns3StartAllCreate(d, meta)
}

func resourceGns3StartAllDelete(d *schema.ResourceData, meta interface{}) error {
	// Intentionally does not stop nodes — see the doc comment on
	// resourceGns3StartAll. This only removes the resource from state.
	d.SetId("")
	return nil
}

func resourceGns3StartAllImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	projectID := d.Id()

	if projectID == "" {
		return nil, fmt.Errorf("missing project_id for gns3_start_all import")
	}

	if err := d.Set("project_id", projectID); err != nil {
		return nil, fmt.Errorf("failed to set project_id: %s", err)
	}

	d.SetId(projectID + "-start")
	return []*schema.ResourceData{d}, nil
}
