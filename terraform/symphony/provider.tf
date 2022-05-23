provider "aws" {
  region = local.region

  default_tags {
    tags = local.default_provider_tags
  }
}

terraform {
  required_providers {
    aws = {
      version = "~> 3.63.0"
    }
  }
  required_version = ">= 0.14"
}
