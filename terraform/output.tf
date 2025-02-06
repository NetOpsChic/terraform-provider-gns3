output "project_id" {
  description = "The ID of the created GNS3 project"
  value       = gns3_project.project1.project_id
}

output "router_ids" {
  description = "The IDs of the created routers"
  value       = { for r in gns3_node.routers : r.name => r.id }
}
