# Create a GNS3 project
resource "gns3_project" "project1" {
  name = var.project_name
}

# Retrieve the template ID for the router
data "gns3_template_id" "router_template" {
  name = var.router_template_name
}

# Create routers based on the provided list
resource "gns3_node" "routers" {
  count       = length(var.routers)
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.template_id
  name        = var.routers[count.index].name
  compute_id  = "local"
  x           = var.routers[count.index].x
  y           = var.routers[count.index].y
}

# Define links to form a square topology
resource "gns3_link" "links" {
  count          = length(var.routers)
  project_id     = gns3_project.project1.project_id
  node_a_id      = element(gns3_node.routers.*.id, count.index)
  node_a_adapter = 4
  node_a_port    = count.index
  node_b_id      = element(gns3_node.routers.*.id, (count.index + 1) % length(var.routers))
  node_b_adapter = 4
  node_b_port    = (count.index + 1) % length(var.routers)
}
