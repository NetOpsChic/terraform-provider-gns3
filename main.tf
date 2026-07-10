terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "2.6.0"
    }
  }
}

provider "gns3" {
  host = "http://localhost:3080"
}

# ── Project ────────────────────────────────────────────────
resource "gns3_project" "project1" {
  name = "test-all-resources-cehck"
}

# ── Template lookup (data source) ─────────────────────────
# Replace "c7200" with a template name that actually exists in your GNS3 install
data "gns3_template_id" "router_template" {
  name = "c7200"
}

# ── QEMU node ──────────────────────────────────────────────
resource "gns3_qemu_node" "csr1" {
  project_id     = gns3_project.project1.project_id
  name           = "CSR1"
  adapter_type   = "virtio-net-pci"
  adapters       = 4
  hda_disk_image = "/home/netopschic/Templates/veos-4.29.2F/hda.qcow2"
  console_type   = "vnc"
  cpus           = 2
  ram            = 4096
  platform       = "x86_64"
  start_vm       = false
  x              = 400
  y              = 100
  symbol         = ":/symbols/qemu_guest.svg"
}

# ── NAT node ───────────────────────────────────────────────
resource "gns3_nat" "nat1" {
  project_id = gns3_project.project1.project_id
  name       = "NAT1"
  x          = 800
  y          = 500
  symbol     = ":/symbols/nat.svg"
}

# ── Switch node ────────────────────────────────────────────
resource "gns3_switch" "switch1" {
  project_id = gns3_project.project1.project_id
  name       = "Switch1"
  compute_id = "local"
  x          = 500
  y          = 300
  symbol = ":/symbols/classic/atm_switch.svg"
}

# ── Cloud node ─────────────────────────────────────────────
resource "gns3_cloud" "cloud1" {
  project_id = gns3_project.project1.project_id
  name       = "Cloud1"
  x          = 500
  y          = 100
  symbol     = ":/symbols/cloud.svg"
}

# ── Template-based node (drag-and-drop device) ─────────────
resource "gns3_template" "router1" {
  project_id  = gns3_project.project1.project_id
  template_id = data.gns3_template_id.router_template.id
  name        = "Router1"
  compute_id  = "local"
  x           = 100
  y           = 100
}

# ── Docker node ─────────────────────────────────────────────
resource "gns3_docker" "docker1" {
  project_id   = gns3_project.project1.project_id
  name         = "Docker1"
  image        = "alpine:latest"
  compute_id   = "local"
  console_type = "none"
  environment  = { FOO = "bar" }
  start        = true
  x            = 100
  y            = 500
  symbol       = ":/symbols/docker_guest.svg"
}

# ── Links — wire switch to nat, cloud, and template router ──
resource "gns3_link" "switch_to_nat" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 3   # changed from 0
  node_b_id      = gns3_nat.nat1.id
  node_b_adapter = 0
  node_b_port    = 0
  depends_on     = [gns3_switch.switch1, gns3_nat.nat1]
}

resource "gns3_link" "switch_to_cloud" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 1
  node_b_id      = gns3_cloud.cloud1.id
  node_b_adapter = 0
  node_b_port    = 0
  depends_on     = [gns3_switch.switch1, gns3_cloud.cloud1]
}

resource "gns3_link" "switch_to_router" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_switch.switch1.id
  node_a_adapter = 0
  node_a_port    = 2
  node_b_id      = gns3_template.router1.id
  node_b_adapter = 0
  node_b_port    = 0
  depends_on     = [gns3_switch.switch1, gns3_template.router1]
}

# ── Start all nodes ──────────────────────────────────────────
resource "gns3_start_all" "start" {
  project_id = gns3_project.project1.project_id
  depends_on = [
    gns3_qemu_node.csr1,
    gns3_nat.nat1,
    gns3_switch.switch1,
    gns3_cloud.cloud1,
    gns3_template.router1,
    gns3_docker.docker1,
  ]
}
# ── Link lookup (data source) — test against switch_to_nat ──
data "gns3_link_id" "switch_to_nat_lookup_reversed" {
  project_id     = gns3_project.project1.project_id
  node_a_id      = gns3_nat.nat1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_switch.switch1.id
  node_b_adapter = 0
  node_b_port    = 3   # updated to match switch_to_nat's new node_a_port
  depends_on     = [gns3_link.switch_to_nat]
}

output "found_link_id_reversed" {
  value = data.gns3_link_id.switch_to_nat_lookup_reversed.link_id
}