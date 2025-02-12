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
# Create GNS3 Project
#############################

resource "gns3_project" "project1" {
  name = "simple-topology"
}

#############################
# Retrieve Router Template
#############################

data "gns3_template_id" "router_template" {
  name = "c7200"  # Replace with your router template name if different.
}

#############################
# Create Network Devices
#############################

# Create a Switch to connect the devices
resource "gns3_switch" "switch1" {
  project_id = gns3_project.project1.project_id
  name       = "Switch1"
  x          = 300
  y          = 300
}

# Create a Cloud node (simulating Internet)
resource "gns3_cloud" "cloud1" {
  project_id = gns3_project.project1.project_id
  name       = "Cloud1"
  x          = 500
  y          = 100
}

# Create a DHCP Server as a Docker container (Ansible ZTP server)
resource "gns3_docker" "dhcp_server" {
  project_id = gns3_project.project1.project_id
  name       = "DHCP_Server"
  compute_id = "local"
  image      = "gns3-dhcp-server"

  environment = {
    DHCP_RANGE   = "192.168.0.223,192.168.0.250,255.255.255.0"
    GATEWAY      = "192.168.0.1"
  }

  x = 500
  y = 300
}

# Create Router 1
resource "gns3_router" "router1" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

# Create Router 2
resource "gns3_router" "router2" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = "Router2"
  compute_id  = "local"
  x           = 100
  y           = 300
}

#############################
# Create Links
#############################

# Connect Router1 to Switch (Router1 port 0 to Switch port 0)
resource "gns3_link" "link_router1_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_router.router1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Connect Router2 to Switch (Router2 port 0 to Switch port 1)
resource "gns3_link" "link_router2_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_router.router2.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 1
}

# Connect Switch to Cloud (Switch port 2 to Cloud port 0)
resource "gns3_link" "link_switch_cloud" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 2
  node_b_id      = gns3_cloud.cloud1.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Connect DHCP Server to Switch (DHCP Server port 0 to Switch port 3)
resource "gns3_link" "link_switch_dhcp" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 3
  node_b_id      = gns3_docker.dhcp_server.id
  node_b_adapter = 0
  node_b_port    = 0

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [gns3_docker.dhcp_server, gns3_switch.switch1]
}

#############################
# Start All Devices (Ensures Everything Boots Up)
#############################

resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.project_id

  depends_on = [
    gns3_router.router1,
    gns3_router.router2,
    gns3_switch.switch1,
    gns3_cloud.cloud1,
    gns3_docker.dhcp_server,
    gns3_link.link_router1_switch,
    gns3_link.link_router2_switch,
    gns3_link.link_switch_cloud,
    gns3_link.link_switch_dhcp
  ]
}
