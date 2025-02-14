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
# Retrieve Template
#############################

data "gns3_template_id" "template" {
  name = "c7200"  # Replace with your template name if different.
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

# Create Template 1 (Router1)
resource "gns3_template" "template1" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.template.template_id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

# Create Template 2 (Router2)
resource "gns3_template" "template2" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.template.template_id
  name        = "Router2"
  compute_id  = "local"
  x           = 100
  y           = 300
}

#############################
# Create Links
#############################

# Connect Template1 to Switch (Router1 port 0 to Switch port 0)
resource "gns3_link" "link_template1_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_template.template1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 0
}

# Connect Template2 to Switch (Router2 port 0 to Switch port 1)
resource "gns3_link" "link_template2_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_template.template2.id
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

#############################
# Start All Devices (Ensures Everything Boots Up)
#############################

resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.project_id

  depends_on = [
    gns3_template.template1,
    gns3_template.template2,
    gns3_switch.switch1,
    gns3_cloud.cloud1,
    gns3_link.link_template1_switch,
    gns3_link.link_template2_switch,
    gns3_link.link_switch_cloud
  ]
}
