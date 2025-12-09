terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "2.5.4"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

resource "gns3_project" "project1" {
  name = "test-import-1"
}

# Cisco CSR1000v (QEMU Node)
resource "gns3_qemu_node" "csr1" {
  project_id     = gns3_project.project1.project_id
  name           = "CSR1"
  adapter_type   = "virtio-net-pci"
  adapters       = 10
  hda_disk_image = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"
  console_type   = "telnet"
  options        = "-nographic -serial mon:stdio"
  cpus           = 3
  ram            = 8194
  mac_address    = "00:1b:54:cc:dd:e1" 
  platform       = "x86_64"
  start_vm       = true

  # Position on GNS3 canvas
  x = 200
  y = 300
}


