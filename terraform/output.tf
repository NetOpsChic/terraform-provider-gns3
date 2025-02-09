output "project_id" {
  description = "The ID of the created GNS3 project"
  value       = gns3_project.project1.project_id
}

output "router_ids" {
  description = "A map of router names to their IDs"
  value       = { for r in gns3_router.routers : r.name => r.id }
}
