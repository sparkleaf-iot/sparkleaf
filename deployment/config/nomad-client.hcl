
client {
  enabled = true
  options {
    "driver.raw_exec.enable"    = "1"
  }
}
acl {
  enabled = true
}

consul {
  address = "127.0.0.1:8500"
  token = "CONSUL_TOKEN"
}

plugin "docker" {
  config {
    allow_privileged = true
  }
}