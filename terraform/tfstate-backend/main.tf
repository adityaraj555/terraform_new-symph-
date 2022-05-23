provider "aws" {
  region = local.region
  default_tags {
    tags = {
      "evtech:environment"     = var.platform_choice
      "evtech:owner"           = "cloudops@eagleview.com"
      "evtech:program"         = "cloudops"
      "evtech:provisioned-by"  = local.provisioned_by_tag
      "evtech:longterm"        = "forever"
      "evtech:terraform-state" = "true"
    }
  }
}

terraform {
  required_providers {
    aws = {
      version = "~> 3.63.0"
    }
  }
}

resource "aws_s3_bucket" "terraform-state" {
  bucket = "${local.resource_name_prefix}-s3bucket-terraform-state"
  # Enable versioning so we can see the full revision history of our
  # state files
  versioning {
    enabled = true
  }
  # Enable server-side encryption by default
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  tags = {
    Name = "${local.resource_name_prefix}:s3bucket-terraform-state"
  }
}

resource "aws_s3_bucket_object" "terraform-state-guard" {
  bucket  = aws_s3_bucket.terraform-state.id
  key     = "terraform-state-guard"
  content = "lock"
}

resource "aws_dynamodb_table" "terraform-locks" {
  name         = "${local.resource_name_prefix}-dynamodbtable-terraform-locks"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"
  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Name = "${local.resource_name_prefix}:dynamodbtable-terraform-locks"
  }
}
