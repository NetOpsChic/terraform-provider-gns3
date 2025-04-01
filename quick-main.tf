terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "2.1.0"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

#############################
# Create GNS3 Project
#############################

resource "gns3_project" "project1" {
  name = "compute-test"
}

#############################
# Create Router1 (QEMU Node)
#############################

resource "gns3_qemu_node" "router1" {
  project_id   = gns3_project.project1.project_id
  name         = "router1"
  adapter_type = "e1000"
  adapters     = 2
  hda_disk_image   = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"  
  console_type = "telnet"
  cpus         = 2
  ram          = 2056
  mac_address  = "00:1c:73:aa:bc:01"
  options      = "-boot order=c -smbios type=1,serial=VEOS1"
  start_vm     = true
  platform     = "x86_64"

  depends_on = [gns3_project.project1]
}

#############################
# Create Router2 (QEMU Node)
#############################

resource "gns3_qemu_node" "router2" {
  project_id   = gns3_project.project1.project_id
  name         = "router2"
  adapter_type = "e1000"
  adapters     = 2
  hda_disk_image   = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"  
  console_type = "telnet"
  cpus         = 2
  ram          = 2056
  mac_address  = "00:1c:73:aa:bc:02"
  options      = "-boot order=c -smbios type=1,serial=VEOS1"
  start_vm     = true
  platform     = "x86_64"

  depends_on = [gns3_project.project1]
}

#############################
# Link Router1 to Router2 (Port 0 of Router1 to Port 0 of Router2)
#############################

resource "gns3_link" "link_router1_router2" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_qemu_node.router1.id
  node_a_adapter = 0  # Adapter 0 of router1
  node_a_port    = 0  # Port 0 of adapter 0
  node_b_id      = gns3_qemu_node.router2.id
  node_b_adapter = 0  # Adapter 0 of router2
  node_b_port    = 0  # Port 0 of adapter 0
}

