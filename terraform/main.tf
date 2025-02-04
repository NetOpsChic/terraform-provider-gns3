terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.0.0"
    }
  }
}

# Configure the GNS3 provider
provider "gns3" {
  host = "http://localhost:3080"
}

# Create a GNS3 project
resource "gns3_project" "project1" {
  name = "My-first-tes-topology"
}

# Retrieve the template ID for the router
data "gns3_template_id" "router_template" {
  name = "c7200"  # Replace with the actual router template name
}

# Define four routers positioned in a square layout
resource "gns3_node" "router1" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

resource "gns3_node" "router2" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router2"
  compute_id  = "local"
  x           = 500
  y           = 100
}

resource "gns3_node" "router3" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router3"
  compute_id  = "local"
  x           = 500
  y           = 500
}

resource "gns3_node" "router4" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router4"
  compute_id  = "local"
  x           = 100
  y           = 500
}

# Define links to form a square topology
resource "gns3_link" "link1" {
  project_id      = gns3_project.project1.project_id
  node_a_id       = gns3_node.router1.id
  node_a_adapter  = 4  # Slot 4
  node_a_port     = 0  # ✅ Port 0
  node_b_id       = gns3_node.router2.id
  node_b_adapter  = 4  # Slot 4
  node_b_port     = 1  # ✅ Port 1
}

resource "gns3_link" "link2" {
  project_id      = gns3_project.project1.project_id
  node_a_id       = gns3_node.router2.id
  node_a_adapter  = 4  # Slot 4
  node_a_port     = 2  # ✅ Port 2
  node_b_id       = gns3_node.router3.id
  node_b_adapter  = 4  # Slot 4
  node_b_port     = 3  # ✅ Port 3
}

resource "gns3_link" "link3" {
  project_id      = gns3_project.project1.project_id
  node_a_id       = gns3_node.router3.id
  node_a_adapter  = 4  # Slot 4
  node_a_port     = 4  # ✅ Port 4
  node_b_id       = gns3_node.router4.id
  node_b_adapter  = 4  # Slot 4
  node_b_port     = 5  # ✅ Port 5
}

resource "gns3_link" "link4" {
  project_id      = gns3_project.project1.project_id
  node_a_id       = gns3_node.router4.id
  node_a_adapter  = 4  # Slot 4
  node_a_port     = 6  # ✅ Port 6
  node_b_id       = gns3_node.router1.id
  node_b_adapter  = 4  # Slot 4
  node_b_port     = 7  # ✅ Port 7
}
