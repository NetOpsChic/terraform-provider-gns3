# Terraform Provider for GNS3

## Overview
The **Terraform Provider for GNS3** allows network engineers and DevOps professionals to automate the deployment and management of **GNS3 network topologies** using Terraform. This provider eliminates manual setup by enabling **Infrastructure as Code (IaC)** for network emulation environments.

## Features
- Create and manage **GNS3 nodes** (Routers, Switches, Cloud and Links).
- Define and configure **network links** between nodes.
- Automate **GNS3 topology deployment** with Terraform.

## Installation
### Prerequisites
- **Terraform** (>= v1.13.0)
- **GNS3 Server** (>= v2.2.0) installed and running
- **GNS3 API** enabled on your server

### Configure the Provider
Add the following to your Terraform configuration:
```hcl
terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.2.0"
    }
  }
}

# Configure the GNS3 provider
provider "gns3" {
  host = "http://localhost:3080"
}
```

### Install the Provider
```bash
terraform init
```

## Files

- `provider.tf`: Configures the GNS3 provider.
- `variables.tf`: Defines input variables.
- `main.tf`: Contains resource definitions.
- `outputs.tf`: Specifies output values.

## Usage 

### Fetching Template_id Router 
```hcl
data "gns3_template_id" "router_template" {
  name = "c7200"  # Replace with the actual router template name
}
```
### Creating a Project
```hcl
resource "gns3_project" "project1" {
  name = "My-first-test-topology"
}
```
### Creating a Router
```hcl
resource "gns3_router" "router1" {
  project_id  = "your_project_id"
  name        = "Router1"
  template    = "cisco_ios"
  x           = 100
  y           = 200
}
```

### Creating a Docker container
```hcl
resource "gns3_docker" "dhcp_server" {
  project_id = gns3_project.project1.project_id
  name       = ""
  compute_id = "local"
  image      = ""

  environment = {
   
  }

  x = 500
  y = 300
}
```
### Creating a Switch
```hcl
resource "gns3_switch" "switch1" {
  project_id = gns3_project.project1.id
  name       = "Switch1"
}
```
### Creating a Cloud
```hcl
resource "gns3_cloud" "cloud1" {
  project_id = gns3_project.project1.id
  name       = "Cloud1"
}
```
### Starting all nodes
```hcl
resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.id
}
```
### Creating a Link Between Nodes
```hcl
resource "gns3_link" "router1_to_switch" {
  project_id = "your_project_id"
  node_a     = gns3_node.router1.id
  node_b     = gns3_node.switch1.id
}
```
## Example Topology (For Quick Spin!)

A **basic topology** connecting a router and a switch:
```hcl
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
```

## Roadmap
- [ ] Improve provider stability and error handling.
- [ ] Add resource for more network devices
- [ ] Enhance state management

## Contributing
Contributions are welcome! To contribute:
1. Fork the repository.
2. Create a new feature branch.
3. Commit your changes.
4. Open a pull request.

## Issues & Feedback
For issues, feature requests, or general discussion, please open a GitHub issue

## License
This project is licensed under the **MIT License**.

---
🚀 **Created & Maintained by [NetOpsChic](https://github.com/netopschic)**
