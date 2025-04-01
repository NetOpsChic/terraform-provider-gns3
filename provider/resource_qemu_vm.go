package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceGns3Qemu defines a new Terraform resource for creating a QEMU VM instance in GNS3.
func resourceGns3Qemu() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3QemuCreate,
		Read:   resourceGns3QemuRead,
		Update: resourceGns3QemuUpdate,
		Delete: resourceGns3QemuDelete,

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
		"console_type": consoleType,
		"ram":          ram,
		"cpus":         cpus,
		"platform":     platform,
	}

	if cdromImage != nil {
		properties["cdrom_image"] = cdromImage.(string)
	}
	if consoleOk {
		properties["console"] = consoleVal.(int)
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
		"properties": properties,
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

	apiURL := fmt.Sprintf("%s/v2/compute/projects/%s/qemu/nodes/%s", config.APIURL, projectID, nodeID)
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
	// Optionally, set additional fields from the API response.
	return nil
}

func resourceGns3QemuUpdate(d *schema.ResourceData, meta interface{}) error {
	// For simplicity, this example does not implement updates.
	return resourceGns3QemuRead(d, meta)
}

func resourceGns3QemuDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	apiURL := fmt.Sprintf("%s/v2/compute/projects/%s/qemu/nodes/%s", config.APIURL, projectID, nodeID)
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
