terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.0.0"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

#############################
# Variables for Cloud & Switch Ports
#############################

# Cloud node always uses adapter 0; specify the port to use.
variable "cloud_port" {
  description = "Port on the cloud node to use for linking."
  type        = number
  default     = 0
}

# Switch port used for connection to the cloud.
variable "switch_cloud_port" {
  description = "Port on the switch to use when connecting to the cloud."
  type        = number
  default     = 0
}

#############################
# Project and Device Resources
#############################

# Create a GNS3 project
resource "gns3_project" "project1" {
  name = "demo"
}

# Data source to retrieve template ID by name
data "gns3_template_id" "router_id" {
  name = "c7200"  # Replace with your router template's name if different.
}

# Create a switch resource
resource "gns3_switch" "switch1" {
  project_id = gns3_project.project1.project_id
  name       = "Switch1"
}

# Create a cloud resource
resource "gns3_cloud" "cloud1" {
  project_id = gns3_project.project1.project_id
  name       = "Cloud1"
}

# Define four routers (renamed from node to router) positioned in a square layout
resource "gns3_router" "router1" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_id.template_id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

resource "gns3_router" "router2" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_id.template_id
  name        = "Router2"
  compute_id  = "local"
  x           = 500
  y           = 100
}

resource "gns3_router" "router3" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_id.template_id
  name        = "Router3"
  compute_id  = "local"
  x           = 500
  y           = 500
}

resource "gns3_router" "router4" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_id.template_id
  name        = "Router4"
  compute_id  = "local"
  x           = 100
  y           = 500
}

#############################
# Revised Link Resources
#############################

# Link: Cloud connects to Switch
resource "gns3_link" "link_cloud_switch" {
  project_id     = gns3_project.project1.project_id
  # Cloud: adapter is always 0; port is specified by var.cloud_port.
  node_a_id      = gns3_cloud.cloud1.id
  node_a_adapter = 0
  node_a_port    = var.cloud_port
  # Switch: for this link, use the port specified by var.switch_cloud_port.
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = var.switch_cloud_port
}

# Link: Switch connects to Router1  
resource "gns3_link" "link_switch_router1" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 1    # Switch port 1 for Router1
  # For Router1, assume its Ethernet interface is on adapter 0, port 0.
  node_b_id      = gns3_router.router1.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Link: Switch connects to Router2  
resource "gns3_link" "link_switch_router2" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 2    # Switch port 2 for Router2
  # For Router2, assume its Ethernet interface is on adapter 0, port 0.
  node_b_id      = gns3_router.router2.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Link: Switch connects to Router3  
resource "gns3_link" "link_switch_router3" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 3    # Switch port 3 for Router3
  # For Router3, assume its Ethernet interface is on adapter 0, port 0.
  node_b_id      = gns3_router.router3.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Link: Switch connects to Router4  
resource "gns3_link" "link_switch_router4" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 4    # Switch port 4 for Router4
  # For Router4, assume its Ethernet interface is on adapter 0, port 0.
  node_b_id      = gns3_router.router4.id
  node_b_adapter = 0
  node_b_port    = 0
}

#############################
# Start All Nodes Resource
#############################

resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.project_id

  depends_on = [
    gns3_cloud.cloud1,
    gns3_switch.switch1,
    gns3_router.router1,
    gns3_router.router2,
    gns3_router.router3,
    gns3_router.router4,
    gns3_link.link_cloud_switch,
    gns3_link.link_switch_router1,
    gns3_link.link_switch_router2,
    gns3_link.link_switch_router3,
    gns3_link.link_switch_router4,
  ]
}
