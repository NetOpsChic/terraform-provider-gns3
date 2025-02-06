variable "gns3_host" {
  description = "The URL of the GNS3 server"
  type        = string
  default     = "http://localhost:3080"
}

variable "project_name" {
  description = "Name of the GNS3 project"
  type        = string
  default     = "My-first-test1-topology"
}

variable "router_template_name" {
  description = "Name of the router template in GNS3"
  type        = string
  default     = "c7200"  # Replace with your actual router template name
}

variable "routers" {
  description = "List of routers with their positions"
  type = list(object({
    name = string
    x    = number
    y    = number
  }))
  default = [
    { name = "Router1", x = 100, y = 100 },
    { name = "Router2", x = 500, y = 100 },
    { name = "Router3", x = 500, y = 500 },
    { name = "Router4", x = 100, y = 500 },
  ]
}
