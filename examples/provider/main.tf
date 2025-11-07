# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    unkey = {
      source = "registry.terraform.io/eyesoft/unkey"
    }
  }
  required_version = ">= 1.1.0"
}

provider "unkey" {
  root_key = "unkey_3Zddr8Te6NRkx4kxrAhZ4y9E"
}

resource "unkey_api" "edu" {
  name = "edu_api"
}

output "edu_order" {
  value = unkey_api.edu
}
