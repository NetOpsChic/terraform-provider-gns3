
# GNS3 Provider for Terraform & OpenTofu

## Overview

The **GNS3 Provider** allows network engineers and DevOps professionals to automate the deployment and management of **GNS3 network topologies**. By enabling **Infrastructure as Code (IaC)**, this provider eliminates manual GUI setup and ensures reproducible network emulation environments.

This provider is officially compatible with both **Terraform** and **OpenTofu**.

## Features

  - Create and manage **GNS3 nodes** (Routers, Switches, Docker containers, Cloud, and Links).
  - Define and configure **network links** between nodes with adapter and port granularity.
  - Automate **GNS3 topology deployment** and lifecycle (Start/Stop).
  - Support for **QEMU nodes** with advanced hardware configuration.

## Installation

### Prerequisites

  - **OpenTofu** (\>= v1.6.0) or **Terraform** (\>= v1.13.0)
  - **GNS3 Server** (\>= v2.2.0) installed and running
  - **GNS3 API** enabled on your server

### Configure the Provider

Add the following to your configuration:

```hcl
terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = ">=2.5.5"
    }
  }
}

# Configure the GNS3 provider
provider "gns3" {
  host = "http://localhost:3080"
}
```

### Install the Provider

If using **OpenTofu**:

```bash
tofu init
```

If using **Terraform**:

```bash
terraform init
```

## Usage

### Fetching Template ID

Templates must be created in the GNS3 GUI first. Use the data source to retrieve the ID by name.

```hcl
data "gns3_template_id" "router_template" {
  name = "c7200" 
}
```

### Creating a Project

```hcl
resource "gns3_project" "lab1" {
  name = "NetOps-Automation-Lab"
}
```

### Creating a QEMU Node

```hcl
resource "gns3_qemu_node" "csr1" {
  project_id     = gns3_project.lab1.id
  name           = "CSR1"
  adapter_type   = "virtio-net-pci"
  adapters       = 10
  hda_disk_image = "/path/to/image.qcow2"
  ram            = 4096
  cpus           = 2
  start_vm       = true
}
```

### Creating a Link

```hcl
resource "gns3_link" "link_1" {
  project_id     = gns3_project.lab1.id
  node_a_id      = gns3_qemu_node.csr1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 1
}
```

## Example Topology

A quick-start configuration to deploy a square topology with four routers:

```hcl
# See /examples directory for full HCL implementation
```

## Registry Status

The GNS3 provider is published and verified on:

  - [**OpenTofu Registry**](https://www.google.com/search?q=https://search.opentofu.org/provider/netopschic/gns3)
  - [**Terraform Registry**](https://www.google.com/search?q=https://registry.terraform.io/providers/netopschic/gns3)

## Roadmap

  - [x] **OpenTofu Verified Registry Support**
  - [ ] Migrate to **Terraform Plugin Framework** for better state management.
  - [ ] Improve provider stability and error handling for large-scale topologies.
  - [ ] Add resource for NAT and Hub devices.

## Contributing

Contributions are welcome\! Please feel free to:

1.  Fork the repository.
2.  Create a new feature branch.
3.  Commit your changes.
4.  Open a pull request.

## Issues & Feedback

For bugs, feature requests, or general discussion, please open a [GitHub Issue](https://www.google.com/search?q=https://github.com/NetOpsChic/terraform-provider-gns3/issues).

## License

This project is licensed under the **MIT License**.

-----

**Created & Maintained by [NetOpsChic](https://github.com/netopschic)**