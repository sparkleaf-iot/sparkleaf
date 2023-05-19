agent { 
    policy = "read"
} 

node { 
    policy = "read" 
} 

namespace "*" { 
    policy = "read" 
    capabilities = ["submit-job", "read-logs", "read-fs", "csi-register-plugin", "csi-write-volume", "csi-read-volume", "csi-mount-volume"]
}