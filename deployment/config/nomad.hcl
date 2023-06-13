datacenter = "dc1"
data_dir = "/opt/nomad"
bind_addr = "0.0.0.0"

advertise {
  http = "{{ GetPublicIP }}"
  rpc  = "{{ GetPublicIP }}"
  serf = "{{ GetPublicIP }}"
}