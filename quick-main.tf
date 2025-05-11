terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "2.5.2"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

resource "gns3_project" "project1" {
  name = "test-read-delete"
}

# ✅ ZTP Template Node (replaces gns3_docker)
data "gns3_template_id" "ztp" {
  name = "ztp-container"
}

resource "gns3_template" "ztp" {
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.ztp.id
  name        = "ztp-server"
  compute_id  = "local"
  start       = true
  x           = 100
  y           = 100
}

# ☁️ Cloud
resource "gns3_cloud" "cloud" {
  name       = "cloud"
  project_id = gns3_project.project1.project_id
}

# 🛜 Management Switch
resource "gns3_switch" "mgmt_switch" {
  name       = "mgmt-switch"
  project_id = gns3_project.project1.project_id
}

# 🧠 Cisco CSR1000v (QEMU Node)
resource "gns3_qemu_node" "csr1" {
  project_id     = gns3_project.project1.project_id
  name           = "CSR1"
  adapter_type   = "virtio-net-pci"
  adapters       = 10
  hda_disk_image = "/home/netopschic/Templates/csr1000vng-universalk9.17.03.05-serial/virtioa.qcow2"
  console_type   = "telnet"
  options        = "-nographic -serial mon:stdio"
  cpus           = 2
  ram            = 8192
  mac_address    = "00:1b:54:cc:dd:ee" 
  platform       = "x86_64"
  start_vm       = true
}

# 🔗 Links

resource "gns3_link" "csr_to_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_qemu_node.csr1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 1
}

resource "gns3_link" "ztp_to_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_template.ztp.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 2
}

resource "gns3_link" "cloud_to_switch" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_cloud.cloud.id
  node_a_adapter = 0
  node_a_port    = 1
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 3
}
