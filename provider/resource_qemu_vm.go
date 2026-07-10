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
				Computed:    true,
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
				Computed:    true,
				Description: "Explicit MAC address to assign to the VM's primary network interface. If unset, GNS3 assigns one automatically.",
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
				Description: "Desired running state of the VM. true = should be running, false = should be stopped. Reconciled on every apply.",
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
			"symbol": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Icon/symbol for the node on the GNS3 canvas (e.g. ':/symbols/qemu_guest.svg'). GNS3 assigns a default if unset. See GET /v2/symbols on your GNS3 server for the full list of available symbol IDs.",
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

	payload := map[string]interface{}{
		"name":         name,
		"node_type":    "qemu",
		"compute_id":   "local",
		"console_type": consoleType,
		"properties":   properties,
	}

	if consoleOk {
		payload["console"] = consoleVal.(int)
	}

	if xv, ok := d.GetOkExists("x"); ok {
		payload["x"] = xv.(int)
	}
	if yv, ok := d.GetOkExists("y"); ok {
		payload["y"] = yv.(int)
	}
	if sv, ok := d.GetOk("symbol"); ok {
		payload["symbol"] = sv.(string)
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
		body, _ := io.ReadAll(resp.Body)
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

	// Reconcile desired running state (start_vm is the source of truth, not
	// "whatever GNS3 happened to do on create" — a freshly created node is
	// stopped by default, so we only need to act when start_vm is true).
	if d.Get("start_vm").(bool) {
		if err := setQemuRunning(http.DefaultClient, config.Host, projectID, nodeID, true); err != nil {
			return err
		}
	}

	return resourceGns3QemuRead(d, meta)
}

func resourceGns3QemuRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to read QEMU node, status: %d, response: %s", resp.StatusCode, body)
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode node details: %s", err)
	}

	if err := d.Set("name", node["name"]); err != nil {
		return err
	}

	if err := setIntField(d, "x", node["x"]); err != nil {
		return err
	}
	if err := setIntField(d, "y", node["y"]); err != nil {
		return err
	}
	if err := setIntField(d, "console", node["console"]); err != nil {
		return err
	}

	if v, ok := node["console_type"].(string); ok && v != "" {
		if err := d.Set("console_type", v); err != nil {
			return err
		}
	}
	if v, ok := node["symbol"].(string); ok && v != "" {
		if err := d.Set("symbol", v); err != nil {
			return err
		}
	}

	if status, ok := node["status"].(string); ok {
		if err := d.Set("start_vm", status == "started"); err != nil {
			return err
		}
	}

	props, _ := node["properties"].(map[string]interface{})
	if props != nil {
		if err := setStringField(d, "adapter_type", props["adapter_type"]); err != nil {
			return err
		}
		if err := setIntField(d, "adapters", props["adapters"]); err != nil {
			return err
		}
		if err := setStringField(d, "bios_image", props["bios_image"]); err != nil {
			return err
		}
		if err := setStringField(d, "cdrom_image", props["cdrom_image"]); err != nil {
			return err
		}
		if err := setIntField(d, "cpus", props["cpus"]); err != nil {
			return err
		}
		if err := setIntField(d, "ram", props["ram"]); err != nil {
			return err
		}
		if err := setStringField(d, "mac_address", props["mac_address"]); err != nil {
			return err
		}
		if err := setStringField(d, "options", props["options"]); err != nil {
			return err
		}
		if err := setStringField(d, "platform", props["platform"]); err != nil {
			return err
		}
		if err := setStringField(d, "hda_disk_image", props["hda_disk_image"]); err != nil {
			return err
		}
	}

	return nil
}

// setIntField sets an int-typed schema field from a decoded JSON value, which
// arrives as float64 (or occasionally int) since encoding/json decodes numbers
// into interface{} as float64 by default. Ignores nil/missing values so we
// don't stomp defaults with zero when the API omits a field.
func setIntField(d *schema.ResourceData, key string, raw interface{}) error {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case float64:
		return d.Set(key, int(v))
	case int:
		return d.Set(key, v)
	}
	return nil
}

// setStringField sets a string-typed schema field from a decoded JSON value,
// skipping nil so empty/unset properties don't overwrite existing state.
func setStringField(d *schema.ResourceData, key string, raw interface{}) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw.(string); ok {
		return d.Set(key, v)
	}
	return nil
}

func resourceGns3QemuUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

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
		d.HasChange("y") ||
		d.HasChange("symbol")) {
		return resourceGns3QemuRead(d, meta)
	}

	client := &http.Client{}

	// 1) GET live node to merge properties & check status
	getURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", config.Host, projectID, nodeID)
	resp, err := http.Get(getURL)
	if err != nil {
		return fmt.Errorf("failed to read QEMU node (pre-update): %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to read QEMU node (pre-update), status: %d, response: %s", resp.StatusCode, string(body))
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode node (pre-update): %s", err)
	}

	props := map[string]interface{}{}
	if p, ok := node["properties"].(map[string]interface{}); ok && p != nil {
		props = p
	}

	// 2) Stop if currently running — some properties require a stopped VM to change.
	wasRunning := false
	if s, ok := node["status"].(string); ok && s == "started" {
		wasRunning = true
		if err := setQemuRunning(client, config.Host, projectID, nodeID, false); err != nil {
			return err
		}
	}
	_ = wasRunning // retained for clarity/logging; final state is driven by start_vm below, not wasRunning

	// 3) Overlay changed fields into properties
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

	// 4) Build PUT payload (top-level name/console/x/y + properties)
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
	if d.HasChange("symbol") {
		if sv, ok := d.GetOk("symbol"); ok {
			putPayload["symbol"] = sv.(string)
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

	putResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update QEMU node: %s", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("update QEMU node failed, status: %d, response: %s", putResp.StatusCode, string(body))
	}

	// 6) Reconcile running state to match the DESIRED state (start_vm), not
	// whatever it happened to be running before the update. This is the fix:
	// previously "wasRunning || start_vm" meant a node the user wanted to stop
	// (start_vm = false) would get silently restarted if it had been running.
	if err := setQemuRunning(client, config.Host, projectID, nodeID, d.Get("start_vm").(bool)); err != nil {
		return err
	}

	// 7) Re-read to sync state
	return resourceGns3QemuRead(d, meta)
}

// setQemuRunning starts or stops a QEMU node via GNS3's node lifecycle endpoints.
// Treats "already in desired state" conflicts (409) as success rather than an error.
func setQemuRunning(client *http.Client, host, projectID, nodeID string, running bool) error {
	action := "stop"
	if running {
		action = "start"
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/%s", host, projectID, nodeID, action)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create %s request: %s", action, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to %s QEMU node: %s", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to %s node, status: %d, response: %s", action, resp.StatusCode, string(body))
	}
	return nil
}

func resourceGns3QemuDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
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

	if err := d.Set("project_id", projectID); err != nil {
		return nil, err
	}
	d.SetId(nodeID)

	return []*schema.ResourceData{d}, nil
}
