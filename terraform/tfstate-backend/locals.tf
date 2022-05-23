locals {
  environment_prefixes = {
    "sandbox" : "sbox",
    "develop" : "dev",
    "prod" : "prod",
    "stage" : "stage",
    "test" : "test"
  }

  environment          = local.environment_prefixes[var.platform_choice]
  version              = "1x0"
  resource_name_prefix = "cops-${local.environment}-${local.version}"
  region               = "us-east-2"
  provisioned_by_tag   = "terraform-cloudops"
}
