acl = "write" 

key_prefix "traefik" {
      policy = "write"
    }
    service "traefik" {
      policy = "write"
    }
agent_prefix "" {
    policy = "write"
} 

event_prefix "" {
    policy = "write"
} 

key_prefix "" {
    policy = "write"
} 

node_prefix "" {
    policy = "write"
} 

query_prefix "" {
    policy = "write"
} 

service_prefix "" {
    policy = "write"
}