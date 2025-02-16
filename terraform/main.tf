# Create a GNS3 project
resource "gns3_project" "project1" {
  name = var.project_name
}

# Retrieve the router template ID using the provided router template name
data "gns3_template_id" "router_id" {
  name = var.router_template_name
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

# Create routers based on the provided list
resource "gns3_template" "routers" {
  count       = length(var.routers)
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_id.template_id
  name        = var.routers[count.index].name
  compute_id  = "local"
  x           = var.routers[count.index].x
  y           = var.routers[count.index].y
}

# Define links between devices
resource "gns3_link" "link_cloud_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_cloud.cloud1.id
  node_a_adapter = 0
  node_a_port    = var.cloud_port
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = var.switch_cloud_port
}

resource "gns3_link" "link_switch_router" {
  count          = length(var.routers)
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = count.index + 1
  node_b_id      = element(gns3_template.routers.*.id, count.index)
  node_b_adapter = 0
  node_b_port    = 0
}

# Start all nodes after they are created
resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.project_id

  depends_on = [
    gns3_cloud.cloud1,
    gns3_switch.switch1,
    gns3_template.routersrouters,
    gns3_link.link_cloud_switch,
    gns3_link.link_switch_router,
  ]
}
