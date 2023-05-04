data_dir = "/opt/consul/data"
bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
advertise_addr = "IP_ADDRESS"

bootstrap_expect = 1

acl {
    enabled = true
    default_policy = "deny"
    down_policy = "extend-cache"
    tokens {
      agent = "BOOTSTRAP_TOKEN"
  }
}

log_level = "INFO"

server = true
ui_config {
  enabled = true
}
retry_join = ["project_name=sparkleaf provider=gce tag_value=auto-join"]

service {
    name = "consul"
}

connect {
  enabled = true
}

ports {
  grpc = 8502
}