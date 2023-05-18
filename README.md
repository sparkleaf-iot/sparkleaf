# sparkleaf

Sparkleaf is about connecting tapping into the plant electrophysiology using ESP32's.

Using Nomad and Consul to deploy container based microservices. Currently only works on GCP.

## Components
### Nomad
Nomad config can be found in deployment/config folder and jobs in deplyoment/jobs.
### Consul
Consul for service discovery, tightly bound to nomad and deployed in a similar fashion.
### Traefik
Traefik as a reverse proxy deployed in nomad using the docker container.
### Pulumi
Pulumi in golang for IaC, see main.go
### Packer
Despite relying on containers for the actual application, a base image is created for vm's that have Consul/Nomad preinstalled. Speeds up boot and scaling. See image.pkr.hcl for details.

## Deployment
Deployment requires a GCP account and credentials and pulumi installed on the local machine. Cd into deplyoment and pulumi up should be the only necessairy step. Consul UI can be found on serverip:8500 and nomad on :4646. Traefik dashboard is on clientip:8081.
