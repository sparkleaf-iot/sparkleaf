server {
  enabled = true
  # This is set to 1 only because it breaks nomad bootstrapping. It should be set to 3 in production.
  bootstrap_expect = 1
}
acl {
  enabled = true
}

consul {
  address = "127.0.0.1:8500"
  token = "CONSUL_TOKEN"
}
