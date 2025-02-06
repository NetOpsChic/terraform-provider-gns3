terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.0.0"
    }
  }
}

provider "gns3" {
  host = var.gns3_host
}
