package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceGns3Qemu defines a new Terraform resource for creating a QEMU VM instance in GNS3.
func resourceGns3Qemu() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3QemuCreate,
		Read:   resourceGns3QemuRead,
		Update: resourceGns3QemuUpdate,
		Delete: resourceGns3QemuDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceQemuImporter, // use custom importer
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UUID of the GNS3 project",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the QEMU VM instance",
			},
			"adapter_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "e1000",
				Description: "QEMU adapter type",
			},
			"adapters": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Number of network adapters",
			},
			"bios_image": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to the QEMU BIOS image",
			},
			"cdrom_image": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to the QEMU CDROM image",
			},
			"console": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Console TCP port",
			},
			"console_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "telnet",
				Description: "Console type (telnet, vnc, spice, etc.)",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Number of vCPUs",
			},
			"ram": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     256,
				Description: "Amount of RAM in MB",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit MAC address to assign to the VM's primary network interface",
			},
			"options": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Additional QEMU options (e.g. -smbios to set serial number)",
			},
			"start_vm": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, start the QEMU VM instance after creation",
			},
			"platform": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Platform architecture for QEMU node (e.g. x86_64, aarch64). Required to determine QEMU binary.",
			},
			"hda_disk_image": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to the HDA (bootable) disk image file for the QEMU node",
			},
			// NEW: optional canvas coordinates
			"x": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "X coordinate of the node on the GNS3 canvas",
			},
			"y": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Y coordinate of the node on the GNS3 canvas",
			},
		},
	}
}

func resourceGns3QemuCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)

	name := d.Get("name").(string)
	adapterType := d.Get("adapter_type").(string)
	adapters := d.Get("adapters").(int)
	biosImage := d.Get("bios_image").(string)
	cdromImage, _ := d.GetOk("cdrom_image")
	consoleVal, consoleOk := d.GetOk("console")
	consoleType := d.Get("console_type").(string)
	cpus := d.Get("cpus").(int)
	ram := d.Get("ram").(int)
	platform := d.Get("platform").(string)

	properties := map[string]interface{}{
		"adapter_type": adapterType,
		"adapters":     adapters,
		"bios_image":   biosImage,
		"cdrom_image":  "",
		"ram":          ram,
		"cpus":         cpus,
		"platform":     platform,
	}

	if cdromImage != nil {
		properties["cdrom_image"] = cdromImage.(string)
	}
	if v, ok := d.GetOk("mac_address"); ok {
		properties["mac_address"] = v.(string)
	}
	if v, ok := d.GetOk("options"); ok {
		properties["options"] = v.(string)
	}
	if v, ok := d.GetOk("hda_disk_image"); ok {
		properties["hda_disk_image"] = v.(string)
		properties["hda_disk_interface"] = "virtio"
	}

	// Controller-level API
	payload := map[string]interface{}{
		"name":       name,
		"node_type":  "qemu",
		"compute_id": "local", // adjust if needed
		"console_type": consoleType,
		"properties": properties,
	}

	if consoleOk {
		payload["console"] = consoleVal.(int)
	}

	// include x/y if explicitly set (even if zero)
	if xv, ok := d.GetOkExists("x"); ok {
		payload["x"] = xv.(int)
	}
	if yv, ok := d.GetOkExists("y"); ok {
		payload["y"] = yv.(int)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal QEMU controller payload: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes", config.Host, projectID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create QEMU node via controller: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("controller rejected QEMU node creation, status: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode controller response: %s", err)
	}

	nodeID, ok := result["node_id"].(string)
	if !ok || nodeID == "" {
		return fmt.Errorf("node_id not returned by controller")
	}
	d.SetId(nodeID)

	// Start VM if requested
	if d.Get("start_vm").(bool) {
		startURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/start", config.Host, projectID, nodeID)
		req, err := http.NewRequest("POST", startURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create start request: %s", err)
		}
		startResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to start QEMU node: %s", err)
		}
		defer startResp.Body.Close()
		if startResp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(startResp.Body)
			return fmt.Errorf("failed to start node, status: %d, response: %s", startResp.StatusCode, string(body))
		}
	}

	return resourceGns3QemuRead(d, meta)
}

func resourceGns3QemuRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// Use the controller's project/node endpoint, not the compute API path
	apiURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", config.Host, projectID, nodeID)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to read QEMU node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	} else if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to read QEMU node, status: %d, response: %s", resp.StatusCode, body)
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode node details: %s", err)
	}

	d.Set("name", node["name"])

	// hydrate x/y if present
	if xv, ok := node["x"]; ok {
		switch t := xv.(type) {
		case float64:
			_ = d.Set("x", int(t))
		case int:
			_ = d.Set("x", t)
		}
	}
	if yv, ok := node["y"]; ok {
		switch t := yv.(type) {
		case float64:
			_ = d.Set("y", int(t))
		case int:
			_ = d.Set("y", t)
		}
	}

	return nil
}

func resourceGns3QemuUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// If nothing changed, just refresh state
	if !(d.HasChange("name") ||
		d.HasChange("adapter_type") ||
		d.HasChange("adapters") ||
		d.HasChange("bios_image") ||
		d.HasChange("cdrom_image") ||
		d.HasChange("console") ||
		d.HasChange("console_type") ||
		d.HasChange("cpus") ||
		d.HasChange("ram") ||
		d.HasChange("mac_address") ||
		d.HasChange("options") ||
		d.HasChange("platform") ||
		d.HasChange("hda_disk_image") ||
		d.HasChange("start_vm") ||
		d.HasChange("x") ||
		d.HasChange("y")) {
		return resourceGns3QemuRead(d, meta)
	}

	// 1) GET live node to merge properties & check status
	getURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", config.Host, projectID, nodeID)
	resp, err := http.Get(getURL)
	if err != nil {
		return fmt.Errorf("failed to read QEMU node (pre-update): %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Node missing -> mark gone so TF can recreate
		d.SetId("")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to read QEMU node (pre-update), status: %d, response: %s", resp.StatusCode, string(body))
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode node (pre-update): %s", err)
	}

	// extract properties map safely
	props := map[string]interface{}{}
	if p, ok := node["properties"].(map[string]interface{}); ok && p != nil {
		props = p
	}

	// 2) Stop if running (some props require stop)
	wasRunning := false
	if s, ok := node["status"].(string); ok && s == "started" {
		wasRunning = true
		stopURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/stop", config.Host, projectID, nodeID)
		req, err := http.NewRequest("POST", stopURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create stop request: %s", err)
		}
		stopResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to stop QEMU node: %s", err)
		}
		defer stopResp.Body.Close()
		if stopResp.StatusCode != http.StatusOK && stopResp.StatusCode != http.StatusConflict {
			body, _ := ioutil.ReadAll(stopResp.Body)
			return fmt.Errorf("failed to stop node, status: %d, response: %s", stopResp.StatusCode, string(body))
		}
	}

	// 3) Overlay changed fields into properties (top-level handled separately)
	if d.HasChange("adapter_type") {
		props["adapter_type"] = d.Get("adapter_type").(string)
	}
	if d.HasChange("adapters") {
		props["adapters"] = d.Get("adapters").(int)
	}
	if d.HasChange("bios_image") {
		props["bios_image"] = d.Get("bios_image").(string)
	}
	if d.HasChange("cdrom_image") {
		if v, ok := d.GetOk("cdrom_image"); ok {
			props["cdrom_image"] = v.(string)
		} else {
			delete(props, "cdrom_image")
		}
	}
	if d.HasChange("cpus") {
		props["cpus"] = d.Get("cpus").(int)
	}
	if d.HasChange("ram") {
		props["ram"] = d.Get("ram").(int)
	}
	if d.HasChange("mac_address") {
		if v, ok := d.GetOk("mac_address"); ok {
			props["mac_address"] = v.(string)
		} else {
			delete(props, "mac_address")
		}
	}
	if d.HasChange("options") {
		if v, ok := d.GetOk("options"); ok {
			props["options"] = v.(string)
		} else {
			delete(props, "options")
		}
	}
	if d.HasChange("platform") {
		props["platform"] = d.Get("platform").(string)
	}
	if d.HasChange("hda_disk_image") {
		if v, ok := d.GetOk("hda_disk_image"); ok {
			props["hda_disk_image"] = v.(string)
			props["hda_disk_interface"] = "virtio"
		} else {
			delete(props, "hda_disk_image")
			delete(props, "hda_disk_interface")
		}
	}

	// 4) Build PUT payload (top-level name/x/y + properties)
	putPayload := map[string]interface{}{
		"properties": props,
	}
	if d.HasChange("name") {
		putPayload["name"] = d.Get("name").(string)
	}
	if d.HasChange("console") {
	  if v, ok := d.GetOk("console"); ok {
		putPayload["console"] = v.(int)
	  }
    }
	if d.HasChange("console_type") {
      putPayload["console_type"] = d.Get("console_type").(string)
    }
	if d.HasChange("x") {
		if xv, ok := d.GetOkExists("x"); ok {
			putPayload["x"] = xv.(int)
		}
	}
	if d.HasChange("y") {
		if yv, ok := d.GetOkExists("y"); ok {
			putPayload["y"] = yv.(int)
		}
	}
	
	// 5) PUT update
	data, err := json.Marshal(putPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %s", err)
	}
	putURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", config.Host, projectID, nodeID)
	req, err := http.NewRequest("PUT", putURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	putResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update QEMU node: %s", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(putResp.Body)
		return fmt.Errorf("update QEMU node failed, status: %d, response: %s", putResp.StatusCode, string(body))
	}

	// 6) Start again if it was running, or if desired state requests it
	if wasRunning || d.Get("start_vm").(bool) {
		startURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/start", config.Host, projectID, nodeID)
		req, err := http.NewRequest("POST", startURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create start request: %s", err)
		}
		startResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to start QEMU node: %s", err)
		}
		defer startResp.Body.Close()
		if startResp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(startResp.Body)
			return fmt.Errorf("failed to start node, status: %d, response: %s", startResp.StatusCode, string(body))
		}
	}

	// 7) Re-read to sync state
	return resourceGns3QemuRead(d, meta)
}

func resourceGns3QemuDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// Use the controller's project/node endpoint for delete as well
	apiURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", config.Host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete QEMU node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete QEMU node, status: %d, response: %s", resp.StatusCode, body)
	}
	d.SetId("")
	return nil
}

// resourceQemuImporter supports both comma- and slash-separated import IDs:
//
//	<node_id>,<project_id>
//	<project_id>/<node_id>
func resourceQemuImporter(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	raw := d.Id()
	var nodeID, projectID string

	if strings.Contains(raw, ",") {
		parts := strings.SplitN(raw, ",", 2)
		nodeID, projectID = parts[0], parts[1]
	} else if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		projectID, nodeID = parts[0], parts[1]
	} else {
		return nil, fmt.Errorf(
			"invalid import ID %q: expected <node_id>,<project_id> or <project_id>/<node_id>",
			raw,
		)
	}

	// seed the required attribute:
	if err := d.Set("project_id", projectID); err != nil {
		return nil, err
	}
	// Terraform resource ID must be the node ID:
	d.SetId(nodeID)

	return []*schema.ResourceData{d}, nil
}
