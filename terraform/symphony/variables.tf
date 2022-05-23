variable "commit_hash" {
  description = "Commit hash used to tag resources"
  default     = ""
}

variable "platform_choice" {
  description = "Platform where the resources will be deployed"
}

variable "build_number" {
  description = "Jenkins build number."
  default     = ""
}

