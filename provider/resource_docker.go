package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DockerProperties holds Docker-specific options for a node.
type DockerProperties struct {
	Image        string   `json:"image"`
	Environment  *string  `json:"environment,omitempty"`
	ExtraVolumes []string `json:"extra_volumes,omitempty"`
	StartCommand *string  `json:"start_command,omitempty"`
}

// DockerNode represents the JSON payload for creating/reading a Docker node.
// console and console_type are top-level Node attributes in the GNS3 API,
// not emulator-specific properties — nesting them under properties works for
// initial creation (GNS3 mirrors the value up to the top level) but updates
// sent under properties are silently ignored, leaving GUI and API state out
// of sync. They must be read and written at the top level.
type DockerNode struct {
	Name        string           `json:"name"`
	NodeType    string           `json:"node_type"`
	ComputeID   string           `json:"compute_id,omitempty"`
	Console     int              `json:"console,omitempty"`
	ConsoleType string           `json:"console_type,omitempty"`
	Symbol      string           `json:"symbol,omitempty"`
	Properties  DockerProperties `json:"properties"`
	NodeID      string           `json:"node_id,omitempty"`
	X           int              `json:"x,omitempty"`
	Y           int              `json:"y,omitempty"`
	Status      string           `json:"status,omitempty"`
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
			"x": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The X coordinate for positioning the Docker node in GNS3 GUI.",
			},
			"y": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The Y coordinate for positioning the Docker node in GNS3 GUI.",
			},
			"console": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Console TCP port. Assigned automatically by GNS3 if not set.",
			},
			"console_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
				Description: "Console type for the Docker node (e.g. none, telnet, vnc).",
			},
			"extra_volumes": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of extra volume mappings in the format 'host_dir:container_dir'.",
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
				Description: "Whether the Docker container should be running. Changing this after creation starts or stops the container.",
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

// buildEnvString converts a Terraform map into GNS3's expected comma-separated
// "KEY=VALUE,KEY2=VALUE2" string. Keys are sorted so the result is deterministic
// and doesn't cause spurious diffs due to Go's randomized map iteration order.
func buildEnvString(envVars map[string]interface{}) string {
	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	envList := make([]string, 0, len(keys))
	for _, k := range keys {
		envList = append(envList, fmt.Sprintf("%s=%s", k, envVars[k].(string)))
	}
	return strings.Join(envList, ",")
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

	var envStr *string
	if v, ok := d.GetOk("environment"); ok {
		formatted := buildEnvString(v.(map[string]interface{}))
		envStr = &formatted
	}

	var extraVolumes []string
	if v, ok := d.GetOk("extra_volumes"); ok {
		for _, vol := range v.([]interface{}) {
			extraVolumes = append(extraVolumes, vol.(string))
		}
	}

	var startCommand *string
	if v, ok := d.GetOk("start_command"); ok {
		cmd := v.(string)
		startCommand = &cmd
	}

	consoleType := d.Get("console_type").(string)
	consoleVal, consoleOk := d.GetOk("console")
	symbol := d.Get("symbol").(string)

	dockerNode := DockerNode{
		Name:        name,
		NodeType:    "docker",
		ComputeID:   computeID,
		ConsoleType: consoleType,
		Symbol:      symbol,
		X:           x,
		Y:           y,
		Properties: DockerProperties{
			Image:        image,
			Environment:  envStr,
			ExtraVolumes: extraVolumes,
			StartCommand: startCommand,
		},
	}
	if consoleOk {
		dockerNode.Console = consoleVal.(int)
	}

	data, err := json.Marshal(dockerNode)
	if err != nil {
		return fmt.Errorf("failed to marshal docker node data: %s", err)
	}

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

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create Docker node, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var createdDocker DockerNode
	if err := json.Unmarshal(body, &createdDocker); err != nil {
		return fmt.Errorf("failed to decode Docker node response: %s", err)
	}

	if createdDocker.NodeID == "" {
		return fmt.Errorf("failed to retrieve node_id from GNS3 API response")
	}

	d.SetId(createdDocker.NodeID)
	if err := d.Set("docker_id", createdDocker.NodeID); err != nil {
		return err
	}

	if d.Get("start").(bool) {
		if err := setDockerRunning(client, host, projectID, createdDocker.NodeID, true); err != nil {
			return err
		}
	}

	return resourceGns3DockerRead(d, meta)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to read Docker node, status code: %d, body: %s", resp.StatusCode, body)
	}

	var node DockerNode
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode Docker node response: %s", err)
	}

	if err := d.Set("name", node.Name); err != nil {
		return err
	}
	if err := d.Set("compute_id", node.ComputeID); err != nil {
		return err
	}
	if err := d.Set("x", node.X); err != nil {
		return err
	}
	if err := d.Set("y", node.Y); err != nil {
		return err
	}
	if err := d.Set("console", node.Console); err != nil {
		return err
	}
	if node.ConsoleType != "" {
		if err := d.Set("console_type", node.ConsoleType); err != nil {
			return err
		}
	}
	if node.Symbol != "" {
		if err := d.Set("symbol", node.Symbol); err != nil {
			return err
		}
	}
	if err := d.Set("docker_id", node.NodeID); err != nil {
		return err
	}
	if node.Properties.Image != "" {
		if err := d.Set("image", node.Properties.Image); err != nil {
			return err
		}
	}
	if err := d.Set("extra_volumes", node.Properties.ExtraVolumes); err != nil {
		return err
	}
	if node.Properties.StartCommand != nil {
		if err := d.Set("start_command", *node.Properties.StartCommand); err != nil {
			return err
		}
	}
	if err := d.Set("start", node.Status == "started"); err != nil {
		return err
	}
	// Note: "environment" is intentionally not synced back here. GNS3 returns
	// it as a single formatted string on read, not as a map, so round-tripping
	// it through d.Set would require re-parsing — left as a follow-up if
	// drift detection on environment vars specifically becomes a priority.

	return nil
}

func resourceGns3DockerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*ProviderConfig)
	host := config.Host
	projectID := d.Get("project_id").(string)
	nodeID := d.Id()

	updateData := map[string]interface{}{}
	properties := map[string]interface{}{}
	hasPropertyChange := false

	if d.HasChange("name") {
		updateData["name"] = d.Get("name").(string)
	}
	if d.HasChange("compute_id") {
		updateData["compute_id"] = d.Get("compute_id").(string)
	}
	if d.HasChange("x") {
		updateData["x"] = d.Get("x").(int)
	}
	if d.HasChange("y") {
		updateData["y"] = d.Get("y").(int)
	}
	if d.HasChange("console") {
		if v, ok := d.GetOk("console"); ok {
			updateData["console"] = v.(int)
		}
	}
	if d.HasChange("console_type") {
		updateData["console_type"] = d.Get("console_type").(string)
	}
	if d.HasChange("symbol") {
		updateData["symbol"] = d.Get("symbol").(string)
	}

	if d.HasChange("environment") {
		envVars := d.Get("environment").(map[string]interface{})
		properties["environment"] = buildEnvString(envVars)
		hasPropertyChange = true
	}
	if d.HasChange("extra_volumes") {
		var extraVolumes []string
		if v, ok := d.GetOk("extra_volumes"); ok {
			for _, vol := range v.([]interface{}) {
				extraVolumes = append(extraVolumes, vol.(string))
			}
		}
		properties["extra_volumes"] = extraVolumes
		hasPropertyChange = true
	}
	if d.HasChange("start_command") {
		properties["start_command"] = d.Get("start_command").(string)
		hasPropertyChange = true
	}

	if hasPropertyChange {
		updateData["properties"] = properties
	}

	if len(updateData) > 0 {
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
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to update Docker node, status code: %d, response: %s", resp.StatusCode, string(body))
		}
	}

	if d.HasChange("start") {
		client := &http.Client{}
		if err := setDockerRunning(client, host, projectID, nodeID, d.Get("start").(bool)); err != nil {
			return err
		}
	}

	return resourceGns3DockerRead(d, meta)
}

// setDockerRunning starts or stops a Docker node via GNS3's node lifecycle endpoints.
func setDockerRunning(client *http.Client, host, projectID, nodeID string, running bool) error {
	action := "stop"
	if running {
		action = "start"
	}

	url := fmt.Sprintf("%s/v2/projects/%s/nodes/%s/%s", host, projectID, nodeID, action)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build %s request: %s", action, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to %s docker node: %s", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to %s docker node, status code: %d, response: %s", action, resp.StatusCode, string(body))
	}

	return nil
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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete docker node, status code: %d, response: %s", resp.StatusCode, string(body))
	}

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
