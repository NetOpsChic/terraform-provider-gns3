terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "2.0.1"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

resource "gns3_project" "project1" {
  name = "netdevop-ztp1"
}

# ZTP Template (Docker Container)
data "gns3_template_id" "ztp" {
  name = "ztp-server"
}

resource "gns3_template" "ztp" {
  name        = "ztp-server"
  project_id  = gns3_project.project1.id
  template_id = data.gns3_template_id.ztp.template_id
  start       = true
}

# Cloud Node
resource "gns3_cloud" "cloud" {
  name       = "cloud"
  project_id = gns3_project.project1.id
}

# Management Switch
resource "gns3_switch" "mgmt_switch" {
  name       = "mgmt-switch"
  project_id = gns3_project.project1.id
}

# QEMU Routers
resource "gns3_qemu_node" "R1" {
  project_id     = gns3_project.project1.id
  name           = "R1"
  adapter_type   = "e1000"
  adapters       = 3
  hda_disk_image = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"
  console_type   = "telnet"
  cpus           = 2
  ram            = 2056
  mac_address    = "00:1c:73:aa:bc:01"
  options        = "-boot order=c -smbios type=1,serial=R1"
  platform       = "x86_64"
  start_vm       = true
}

resource "gns3_qemu_node" "R2" {
  project_id     = gns3_project.project1.id
  name           = "R2"
  adapter_type   = "e1000"
  adapters       = 3
  hda_disk_image = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"
  console_type   = "telnet"
  cpus           = 2
  ram            = 2056
  mac_address    = "00:1c:73:aa:bc:02"
  options        = "-boot order=c -smbios type=1,serial=R2"
  platform       = "x86_64"
  start_vm       = true
}

resource "gns3_qemu_node" "R3" {
  project_id     = gns3_project.project1.id
  name           = "R3"
  adapter_type   = "e1000"
  adapters       = 3
  hda_disk_image = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"
  console_type   = "telnet"
  cpus           = 2
  ram            = 2056
  mac_address    = "00:1c:73:aa:bc:03"
  options        = "-boot order=c -smbios type=1,serial=R3"
  platform       = "x86_64"
  start_vm       = true
}

# Router-to-Router Links
resource "gns3_link" "R1_to_R2" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_qemu_node.R1.id
  node_a_adapter = 1
  node_a_port    = 0
  node_b_id      = gns3_qemu_node.R2.id
  node_b_adapter = 1
  node_b_port    = 0
}

resource "gns3_link" "R2_to_R3" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_qemu_node.R2.id
  node_a_adapter = 2
  node_a_port    = 0
  node_b_id      = gns3_qemu_node.R3.id
  node_b_adapter = 1
  node_b_port    = 0
}

# Management Links (Routers to mgmt-switch)
resource "gns3_link" "R1_to_switch" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_qemu_node.R1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 3
}

resource "gns3_link" "R2_to_switch" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_qemu_node.R2.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 4
}

resource "gns3_link" "R3_to_switch" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_qemu_node.R3.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 5
}

# ZTP to mgmt-switch
resource "gns3_link" "ZTP_to_switch" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_template.ztp.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 1
}

# Cloud to mgmt-switch
resource "gns3_link" "Cloud_to_switch" {
  project_id     = gns3_project.project1.id
  node_a_id      = gns3_cloud.cloud.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.mgmt_switch.id
  node_b_adapter = 0
  node_b_port    = 2
}
# Start all
resource "gns3_start_all" "start_nodes" {
  project_id = gns3_project.project1.id
  depends_on = [
    gns3_qemu_node.R1,
    gns3_qemu_node.R2,
    gns3_qemu_node.R3,
    gns3_template.ztp,
    gns3_switch.mgmt_switch,
    gns3_cloud.cloud
  ]
}