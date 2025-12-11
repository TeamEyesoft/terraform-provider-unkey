# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    unkey = {
      source = "eyesoft/unkey"
    }
  }
  required_version = ">= 1.1.0"
}

provider "unkey" {
  root_key = "unkey_XXX"
}

resource "unkey_api" "demo" {
  name = "demo-api"
}

resource "unkey_role" "role_admin" {
  name        = "admin-role"
  description = "Administrator role with full access"
}

resource "unkey_permission" "read_documents" {
  name        = "Read documents"
  slug        = "read.documents"
  description = "Permission to read documents"
}

# Missing a way to link permissions to roles via terraform

resource "unkey_key" "admin_key" {
  api_id      = unkey_api.demo.id
  name        = "demo-key"
  byte_length = 32
  roles       = [unkey_role.role_admin.name]
  permissions = [unkey_permission.read_documents.slug]
  enabled     = true
  meta = jsonencode({
    user_id = "user-12345"
  })
}

output "demo_order" {
  value = unkey_api.demo
}
