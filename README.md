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
      version = "2.0.0"
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
  name = "c7200"  # Replace with the actual template name
}
```
### Creating a Project
```hcl
resource "gns3_project" "project1" {
  name = "My-first-test-topology"
}
```
### Creating a router or any device from template. Devices which are configured in gns3 can be deployed using this resource.
```hcl
# Previous configuration
resource "gns3_router" "router1" {
  # resource parameters
}

# Updated configuration
resource "gns3_template" "router1" {
  # resource parameters
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
terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.1.0"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

# Create a GNS3 project
resource "gns3_project" "project1" {
  name = "simple-topology"
}

# Retrieve the template ID for the router
data "gns3_template_id" "router_template" {
  name = "c7200"  # Replace with your actual router template name
}

# Create a switch
resource "gns3_switch" "switch1" {
  project_id = gns3_project.project1.id
  name       = "Switch1"
  x          = 300
  y          = 300
}

# Create a cloud node
resource "gns3_cloud" "cloud1" {
  project_id = gns3_project.project1.id
  name       = "Cloud1"
  x          = 500
  y          = 100
}

# Create four routers positioned in a square layout
resource "gns3_template" "router1" {
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.router_template.id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

resource "gns3_template" "router2" {
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.router_template.id
  name        = "Router2"
  compute_id  = "local"
  x           = 500
  y           = 100
}

resource "gns3_template" "router3" {
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.router_template.id
  name        = "Router3"
  compute_id  = "local"
  x           = 500
  y           = 500
}

resource "gns3_template" "router4" {
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.router_template.id
  name        = "Router4"
  compute_id  = "local"
  x           = 100
  y           = 500
}

# Define links to form a square topology
resource "gns3_link" "link1" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_template.router1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_template.router2.id
  node_b_adapter = 0
  node_b_port    = 0
}

resource "gns3_link" "link2" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_template.router2.id
  node_a_adapter = 0
  node_a_port    = 1
  node_b_id      = gns3_template.router3.id
  node_b_adapter = 0
  node_b_port    = 1
}

resource "gns3_link" "link3" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_template.router3.id
  node_a_adapter = 0
  node_a_port    = 2
  node_b_id      = gns3_template.router4.id
  node_b_adapter = 0
  node_b_port    = 2
}

resource "gns3_link" "link4" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_template.router4.id
  node_a_adapter = 0
  node_a_port    = 3
  node_b_id      = gns3_template.router1.id
  node_b_adapter = 0
  node_b_port    = 3
}

# Start all devices
resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.id

  depends_on = [
    gns3_template.router1,
    gns3_template.router2,
    gns3_template.router3,
    gns3_template.router4,
    gns3_switch.switch1,
    gns3_cloud.cloud1,
    gns3_link.link1,
    gns3_link.link2,
    gns3_link.link3,
    gns3_link.link4,
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
