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

// DockerProperties holds Docker-specific options for a node.
type DockerProperties struct {
	Image        string   `json:"image"`
	Environment  *string  `json:"environment,omitempty"`
	ConsoleType  string   `json:"console_type"`
	ExtraVolumes []string `json:"extra_volumes,omitempty"`
	StartCommand *string  `json:"start_command,omitempty"`
}

// DockerNode represents the JSON payload for creating a Docker node.
type DockerNode struct {
	Name       string           `json:"name"`
	NodeType   string           `json:"node_type"`
	ComputeID  string           `json:"compute_id,omitempty"`
	Properties DockerProperties `json:"properties"`
	NodeID     string           `json:"node_id,omitempty"`
	X          int              `json:"x,omitempty"` // Added X coordinate
	Y          int              `json:"y,omitempty"` // Added Y coordinate
}

func resourceGns3Docker() *schema.Resource {
	return &schema.Resource{
		Create: resourceGns3DockerCreate,
		Read:   resourceGns3DockerRead,
		Update: resourceGns3DockerUpdate,
		Delete: resourceGns3DockerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGns3DockerImporter,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project ID where the Docker node will be created.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Docker node.",
			},
			"compute_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "local",
				Description: "The compute ID (default: 'local').",
			},
			"image": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // Ensures re-creation when image changes
				Description: "The Docker image name. The image must be available in GNS3.",
			},
			"environment": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Optional Docker environment variables in key-value format.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"x": { // Added X coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The X coordinate for positioning the Docker node in GNS3 GUI.",
			},
			"y": { // Added Y coordinate support
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The Y coordinate for positioning the Docker node in GNS3 GUI.",
			},
			"extra_volumes": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of extra volume mappings in the format 'host_dir:container_dir'. This will be passed inside the properties.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"docker_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier for the Docker node returned by the API.",
			},
			"start_command": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Command to run when starting the Docker container.",
			},
			"start": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to start the Docker container after creation.",
			},
		},
	}
}

func resourceGns3DockerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)
	computeID := d.Get("compute_id").(string)
	image := d.Get("image").(string)
	x := d.Get("x").(int)
	y := d.Get("y").(int)

	// Convert environment map into a single string format (comma-separated key=value pairs)
	var envStr *string
	if v, ok := d.GetOk("environment"); ok {
		envVars := v.(map[string]interface{})
		envList := []string{}
		for key, value := range envVars {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value.(string)))
		}
		envFormatted := strings.Join(envList, ",")
		envStr = &envFormatted
	}

	// Retrieve extra volumes if provided
	var extraVolumes []string
	if v, ok := d.GetOk("extra_volumes"); ok {
		for _, vol := range v.([]interface{}) {
			extraVolumes = append(extraVolumes, vol.(string))
		}
	}

	// Retrieve optional start_command
	var startCommand *string
	if v, ok := d.GetOk("start_command"); ok {
		cmd := v.(string)
		startCommand = &cmd
	}

	// Build the payload for the Docker node
	dockerNode := DockerNode{
		Name:      name,
		NodeType:  "docker",
		ComputeID: computeID,
		X:         x,
		Y:         y,
		Properties: DockerProperties{
			Image:        image,
			Environment:  envStr,
			ConsoleType:  "none",
			ExtraVolumes: extraVolumes,
			StartCommand: startCommand,
		},
	}

	// Marshal the request
	data, err := json.Marshal(dockerNode)
	if err != nil {
		return fmt.Errorf("failed to marshal docker node data: %s", err)
	}

	// Create node via API
	url := fmt.Sprintf("%s/v2/projects/%s/nodes", host, projectID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create Docker node, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Parse created response
	var createdDocker DockerNode
	if err := json.Unmarshal(body, &createdDocker); err != nil {
		return fmt.Errorf("failed to decode Docker node response: %s", err)
	}

	if createdDocker.NodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	// Save ID
	d.SetId(createdDocker.NodeID)
	d.Set("docker_id", createdDocker.NodeID)

	// Optionally start the container
	if d.Get("start").(bool) {
		startURL := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/start", host, projectID, createdDocker.NodeID)
		startReq, err := http.NewRequest("POST", startURL, nil)
		if err != nil {
			return fmt.Errorf("failed to build start request: %s", err)
		}
		startResp, err := client.Do(startReq)
		if err != nil {
			return fmt.Errorf("failed to start docker node: %s", err)
		}
		defer startResp.Body.Close()

		if startResp.StatusCode != http.StatusOK {
			startBody, _ := ioutil.ReadAll(startResp.Body)
			return fmt.Errorf("failed to start docker node, status code: %d, response: %s", startResp.StatusCode, string(startBody))
		}
	}

	return nil
}

func resourceGns3DockerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to retrieve Docker node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read Docker node, status code: %d", resp.StatusCode)
	}

	// Optionally, you can decode the response to update state further.
	return nil
}

func resourceGns3DockerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	// Build the updated payload.
	updateData := make(map[string]interface{})
	if d.HasChange("environment") {
		envVars := d.Get("environment").(map[string]interface{})
		envList := []string{}
		for key, value := range envVars {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value.(string)))
		}
		envFormatted := strings.Join(envList, ",")
		updateData["environment"] = envFormatted
	}
	// Note: Image is ForceNew so we do not update it.
	// Also, extra_volumes, x, and y are typically not updated dynamically, but you could add them if needed.

	data, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %s", err)
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create update request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update Docker node: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to update Docker node, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	if d.HasChange("start_command") {
		updateData["start_command"] = d.Get("start_command").(string)
	}

	return resourceGns3DockerRead(d, meta)
}

func resourceGns3DockerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s", host, projectID, nodeID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for docker node: %s", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete docker node: %s", err)
	}
	defer resp.Body.Close()

	d.SetId("")
	return nil
}
func resourceGns3DockerImporter(
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
