server {
  enabled = true
  bootstrap_expect = 1
}
acl {
  enabled = true
}

consul {
  address = "127.0.0.1:8500"
  token = "CONSUL_TOKEN"
}
