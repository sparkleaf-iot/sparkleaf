ui = true
log_level = "INFO"
data_dir = "/opt/consul/data"
bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
advertise_addr = "IP_ADDRESS"
retry_join = ["project_name=sparkleaf provider=gce tag_value=auto-join"]

acl {
    enabled = true
    default_policy = "deny"
    down_policy = "extend-cache"
     tokens {
      agent = "BOOTSTRAP_TOKEN"
    }
}

connect {
  enabled = true
}
ports {
  grpc = 8502
}