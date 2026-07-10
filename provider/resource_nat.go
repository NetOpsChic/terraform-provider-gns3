package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Nat represents a GNS3 nat node API request/response.
type Nat struct {
	Name      string `json:"name"`
	NodeType  string `json:"node_type"`
	ComputeID string `json:"compute_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
}

func resourceGns3Nat() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3NatCreate,
		Read:   resourceGns3NatRead,
		Update: resourceGns3NatUpdate,
		Delete: resourceGns3NatDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3NatImporter,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project ID where the nat node is deployed.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the nat node. Note: GNS3 does not support renaming NAT nodes in place via the API — changing this value will destroy and recreate the resource.",
			},
			"compute_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "local",
				Description: "Compute ID where the nat node is running.",
			},
			"x": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "X position of the nat node in GNS3 GUI.",
			},
			"y": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Y position of the nat node in GNS3 GUI.",
			},
			"nat_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The nat node's ID assigned by GNS3.",
			},
			"symbol": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Icon/symbol for the node on the GNS3 canvas (e.g. ':/symbols/computer.svg'). GNS3 assigns a default if unset. See GET /v2/symbols on your GNS3 server for the full list of available symbol IDs.",
			},
		},
	}
}

func resourceGns3NatCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	x := d.Get("x").(int)
	y := d.Get("y").(int)
	symbol := d.Get("symbol").(string)

	nat := Nat{
		Name:      name,
		NodeType:  "nat",
		ComputeID: computeID,
		X:         x,
		Y:         y,
		Symbol:    symbol,
	}

	data, err := json.Marshal(nat)
	if err != nil {
		return fmt.Errorf("failed to marshal nat node data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating GNS3 nat node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create nat node, status code: %d, error: %v", resp.StatusCode, errResp)
	}

	var createdNat Nat
	if err := json.NewDecoder(resp.Body).Decode(&createdNat); err != nil {
		return fmt.Errorf("failed to decode nat node response: %s", err)
	}

	if createdNat.NodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	d.SetId(createdNat.NodeID)
	if err := d.Set("nat_id", createdNat.NodeID); err != nil {
		return err
	}
	return nil
}

// Update function for modifying existing nat nodes
func resourceGns3NatUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	natID := d.Id()

	updateData := map[string]interface{}{}

	// Note: "name" is ForceNew — GNS3 silently ignores rename requests for
	// nat nodes, so Terraform recreates the resource instead of calling Update.

	if d.HasChange("compute_id") {
		updateData["compute_id"] = d.Get("compute_id").(string)
	}

	if d.HasChange("x") {
		updateData["x"] = d.Get("x").(int)
	}

	if d.HasChange("y") {
		updateData["y"] = d.Get("y").(int)
	}

	if d.HasChange("symbol") {
		updateData["symbol"] = d.Get("symbol").(string)
	}

	if len(updateData) == 0 {
		return nil
	}

	updateBody, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, natID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(updateBody))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error updating GNS3 nat node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update nat node, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resourceGns3NatRead(d, meta)
}

func resourceGns3NatRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error reading nat node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Node no longer exists in GNS3 — mark resource as gone
		d.SetId("")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected read status %d: %s", resp.StatusCode, body)
	}

	var nat Nat
	if err := json.NewDecoder(resp.Body).Decode(&nat); err != nil {
		return fmt.Errorf("failed to decode nat node response: %s", err)
	}

	if err := d.Set("name", nat.Name); err != nil {
		return err
	}
	if err := d.Set("compute_id", nat.ComputeID); err != nil {
		return err
	}
	if err := d.Set("x", nat.X); err != nil {
		return err
	}
	if err := d.Set("y", nat.Y); err != nil {
		return err
	}
	if err := d.Set("nat_id", nat.NodeID); err != nil {
		return err
	}
	if err := d.Set("symbol", nat.Symbol); err != nil {
		return err
	}

	return nil
}

func resourceGns3NatDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for nat node: %s", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete nat node: %s", err)
	}
	defer resp.Body.Close()

	// Treat "already gone" as success so destroy doesn't fail on drift
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to delete nat node, status code: %d", resp.StatusCode)
	}

	d.SetId("")
	return nil
}

func resourceGns3NatImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	raw := d.Id()
	var projectID, nodeID string

	if parts := strings.SplitN(raw, "/", 2); len(parts) == 2 {
		projectID = parts[0]
		nodeID = parts[1]
	} else {
		return nil, fmt.Errorf("invalid import ID %q — expected format <project_id>/<node_id>", raw)
	}

	if err := d.Set("project_id", projectID); err != nil {
		return nil, err
	}
	d.SetId(nodeID)

	return []*schema.ResourceData{d}, nil
}
