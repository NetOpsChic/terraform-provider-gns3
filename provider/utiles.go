package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Fetch the first available project ID (used by both nodes and links)
func getProjectID(host string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v2/projects", host))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var projects []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return "", err
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no GNS3 projects found")
	}
	return projects[0]["project_id"].(string), nil
}

// Function to get template ID from template name
func getTemplateID(host string, templateName string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v2/templates", host))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var templates []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		return "", err
	}

	for _, template := range templates {
		if template["name"].(string) == templateName {
			if id, ok := template["template_id"].(string); ok {
				return id, nil
			} else if id, ok := template["id"].(string); ok {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("template %s not found", templateName)
}