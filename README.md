# GNS3 Provider for Terraform & OpenTofu

## Overview

The **GNS3 Provider** allows network engineers and DevOps professionals to automate the deployment and management of **GNS3 network topologies**. By enabling **Infrastructure as Code (IaC)**, this provider eliminates manual GUI setup and ensures reproducible network emulation environments.

This provider is officially compatible with both **Terraform** and **OpenTofu**.

## Features

  - Create and manage **GNS3 nodes** (Routers, Switches, NAT, Docker containers, Cloud, and Links).
  - Define and configure **network links** between nodes with adapter and port granularity.
  - Automate **GNS3 topology deployment** and lifecycle (Start/Stop).
  - Support for **QEMU nodes** with advanced hardware configuration.
  - Set a custom **canvas icon (`symbol`)** on any node, matching GNS3's own symbol IDs.
  - Full drift detection — node position, name, and configuration changes made outside Terraform (e.g. in the GNS3 GUI) are correctly detected on `plan`.

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
      version = ">=2.6.0"
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

### Creating a NAT node

> **Note:** Renaming a NAT node after creation is not supported by the GNS3 API — this is a confirmed upstream limitation ([GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811)), not a bug in this provider. Changing `name` will force recreation of the resource instead of silently failing to apply.

```hcl
resource "gns3_nat" "wan_gateway" {
  project_id = gns3_project.lab1.id
  name       = "Internet Gateway"
  x          = -100
  y          = -200
  symbol     = ":/symbols/computer.svg"
}
```

### Creating a Switch node

```hcl
resource "gns3_switch" "switch1" {
  project_id = gns3_project.lab1.id
  name       = "Switch1"
  compute_id = "local"
  x          = 500
  y          = 300
  symbol     = ":/symbols/classic/atm_switch.svg"
}
```

### Creating a Cloud node

```hcl
resource "gns3_cloud" "internet" {
  project_id = gns3_project.lab1.id
  name       = "Cloud1"
  x          = 500
  y          = 100
  symbol     = ":/symbols/cloud.svg"
}
```

### Creating a Docker node

```hcl
resource "gns3_docker" "alpine1" {
  project_id   = gns3_project.lab1.id
  name         = "Alpine1"
  image        = "alpine:latest"
  compute_id   = "local"
  console_type = "telnet"
  environment  = { FOO = "bar" }
  start        = true
  x            = 100
  y            = 500
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

> **Note:** GNS3 does not support rewiring an existing link's endpoints in place. Changing any of `node_a_id`, `node_a_adapter`, `node_a_port`, `node_b_id`, `node_b_adapter`, or `node_b_port` will destroy and recreate the link.

### Looking up an existing Link

```hcl
data "gns3_link_id" "existing_link" {
  project_id     = gns3_project.lab1.id
  node_a_id      = gns3_qemu_node.csr1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 1
}
```
## Known Limitations

These are confirmed GNS3 API behaviors, not bugs in this provider:

  - **NAT node rename is not supported by GNS3's API.** The API accepts the request and returns success, but the name is never actually changed server-side. Tracked upstream: [GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811). Changing `gns3_nat.name` forces recreation.
  - **Link endpoints cannot be rewired in place.** GNS3 accepts a `PUT` request to change an existing link's connected nodes/adapters/ports but does not apply it. All endpoint fields on `gns3_link` force recreation instead.
  - **`gns3_start_all` has no drift detection and does not stop nodes on destroy.** It models a one-time "start everything" action rather than an ongoing state GNS3 exposes for querying, so Terraform cannot detect if nodes are later stopped outside of Terraform, and `terraform destroy` on this resource only removes it from state — it does not stop your lab.
  - **`gns3_docker`'s `environment` variables are not read back from GNS3.** The API returns them as a single flattened string rather than a map, so drift on individual environment variables is not currently detected.

## Registry Status

The GNS3 provider is published and verified on:

  - [**OpenTofu Registry**](https://search.opentofu.org/provider/netopschic/gns3)
  - [**Terraform Registry**](https://registry.terraform.io/providers/netopschic/gns3)

## Roadmap

  - [x] **OpenTofu Verified Registry Support**
  - [x] Custom node icon (`symbol`) support
  - [ ] Migrate to **Terraform Plugin Framework** for better state management.
  - [ ] Improve provider stability and error handling for large-scale topologies.
  - [ ] Support uploading custom symbols/icons (`POST /v2/symbols/{symbol_id}/raw`), rather than only referencing existing ones.
  - [ ] `environment` drift detection for `gns3_docker`.

## Contributing

Contributions are welcome\! Please feel free to:

1.  Fork the repository.
2.  Create a new feature branch.
3.  Commit your changes.
4.  Open a pull request.

## Issues & Feedback

For bugs, feature requests, or general discussion, please open a [GitHub Issue](https://github.com/NetOpsChic/terraform-provider-gns3/issues).

## License

This project is licensed under the **MIT License**.

-----

**Created & Maintained by [NetOpsChic](https://github.com/netopschic)**