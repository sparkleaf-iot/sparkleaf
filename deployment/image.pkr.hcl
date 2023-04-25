locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
}

variable "project" {
  type = string
}

variable "zone" {
  type = string
}

source "googlecompute" "sparkleaf" {
  image_name   = "sparkleaf-${local.timestamp}"
  project_id   = var.project
  source_image = "ubuntu-minimal-2204-jammy-v20230420"
  ssh_username = "packer"
  zone         = var.zone
}

build {
  sources = ["sources.googlecompute.sparkleaf"]

  provisioner "shell" {
    inline = ["sudo mkdir -p /ops/config", "sudo chmod 777 -R /ops"]
  }

  provisioner "file" {
    destination = "/ops"
    source      = "./config"
  }

  provisioner "shell" {
    environment_vars = ["INSTALL_NVIDIA_DOCKER=false", "CLOUD_ENV=gce"]
    script           = "./config/setup.sh"
  }
}