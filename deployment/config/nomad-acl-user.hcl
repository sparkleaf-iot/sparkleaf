agent { 
    policy = "read"
} 

node { 
    policy = "read" 
} 
plugin {
  policy = "read"
}

host_volume "*" {
  policy = "write"
}

namespace "*" { 
    policy = "write" 
    capabilities = ["submit-job", "read-logs", "read-fs", "csi-register-plugin", "csi-write-volume", "csi-read-volume", "csi-mount-volume"]
}