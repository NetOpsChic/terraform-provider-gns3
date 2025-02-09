terraform {
  required_providers {
    gns3 = {
      source  = "netopschic/gns3"
      version = "1.1.0"
    }
  }
}

# Configure the GNS3 provider
provider "gns3" {
  host = "http://localhost:3080"
}