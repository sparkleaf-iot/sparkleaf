package main

import (
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func readFileOrPanic(path string, ctx *pulumi.Context) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}

	// return pulumi.String(string(data))
	return string(data)
}

func injectToken(token string, toBeReplaced string, script string, amount int) string {

	return strings.Replace(script, toBeReplaced, token, amount)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the configuration values from the appropriate yaml.
		gcpConf := config.New(ctx, "gc")
		machineImage := gcpConf.Require("machine_image")
		// Create a bootstrap token for Consul and Nomad
		nomad_consul_token_id := uuid.NewString()
		nomad_consul_token_secret := uuid.NewString()

		// Create a new VPC network for the Nomad server.
		network, err := compute.NewNetwork(ctx, "nomad-network", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		// Create a firewall rule to allow traffic to the Nomad server.
		firewall, err := compute.NewFirewall(ctx, "nomad-firewall", &compute.FirewallArgs{
			Network: network.SelfLink,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("icmp"),
				},

				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("22"),
						pulumi.String("4646"),
						pulumi.String("8500"),
						pulumi.String("80"),
					},
				},
			},
			SourceRanges: pulumi.StringArray{
				pulumi.String("0.0.0.0/0"),
			},
		})
		if err != nil {
			return err
		}

		// Create firewall to allow all internal traffic

		internalFirewall, err := compute.NewFirewall(ctx, "internal-firewall", &compute.FirewallArgs{
			Network: network.SelfLink,
			SourceTags: pulumi.StringArray{
				pulumi.String("auto-join"),
			},
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("icmp"),
				},
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("0-65535"),
					},
				},
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("udp"),
					Ports: pulumi.StringArray{
						pulumi.String("0-65535"),
					},
				},
			},
		})

		if err != nil {
			return err
		}

		serverStartupScript := readFileOrPanic("config/server.sh", ctx)
		serverStartupScript = injectToken(nomad_consul_token_id, "nomad_consul_token_id", serverStartupScript, 1)
		serverStartupScript = injectToken(nomad_consul_token_secret, "nomad_consul_token_secret", serverStartupScript, 2)
		ctx.Log.Info(serverStartupScript, nil)
		// Create a new GCP compute instance to run the Nomad servers on.
		server, err := compute.NewInstance(ctx, "nomad-server", &compute.InstanceArgs{
			MachineType:            pulumi.String("e2-micro"),
			Zone:                   pulumi.String("europe-central2-a"),
			MetadataStartupScript:  pulumi.String(serverStartupScript),
			AllowStoppingForUpdate: pulumi.Bool(true),
			Tags:                   pulumi.StringArray{pulumi.String("auto-join")},
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String(machineImage),
				},
			},
			NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network: network.ID(),
					AccessConfigs: &compute.InstanceNetworkInterfaceAccessConfigArray{
						&compute.InstanceNetworkInterfaceAccessConfigArgs{},
					},
					// Subnetwork: subnet.SelfLink,
				},
			},
			ServiceAccount: &compute.InstanceServiceAccountArgs{
				Scopes: pulumi.StringArray{
					pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{firewall, internalFirewall}))
		if err != nil {
			return err
		}

		clientStartupScript := readFileOrPanic("config/client.sh", ctx)
		//clientStartupScript = injectToken(bootstrap_token, clientStartupScript)

		// Create a new GCP compute instance to run the Nomad cleints on.
		client, err := compute.NewInstance(ctx, "nomad-client", &compute.InstanceArgs{
			MachineType:            pulumi.String("e2-micro"),
			Zone:                   pulumi.String("europe-central2-a"),
			MetadataStartupScript:  pulumi.String(clientStartupScript),
			AllowStoppingForUpdate: pulumi.Bool(true),
			Tags:                   pulumi.StringArray{pulumi.String("auto-join")},
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String(machineImage),
				},
			},
			NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network: network.ID(),
					AccessConfigs: &compute.InstanceNetworkInterfaceAccessConfigArray{
						&compute.InstanceNetworkInterfaceAccessConfigArgs{},
					},
				},
			},
			ServiceAccount: &compute.InstanceServiceAccountArgs{
				Scopes: pulumi.StringArray{
					pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{firewall, internalFirewall}))

		if err != nil {
			return err
		}

		// traefikJob, err := nomad.NewJob(ctx, "traefik-cluster", &nomad.JobArgs{
		// 	Jobspec: readFileOrPanic("jobs/traefik.nomad.hcl", ctx),
		// })

		// if err != nil {
		// 	return err
		// }

		// influxJob, err := nomad.NewJob(ctx, "influx-cluster", &nomad.JobArgs{
		// 	Jobspec: readFileOrPanic("jobs/influx.nomad.hcl", ctx),
		// }, pulumi.Provider(provider))

		// if err != nil {
		// 	return err
		// }

		// ctx.Export("traefikJob", traefikJob.ID())
		// ctx.Export("influxJob", influxJob.ID())
		ctx.Export("server", server.Name)
		ctx.Export("serverIP", server.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
		ctx.Export("client", client.Name)
		ctx.Export("clientIP", client.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
		ctx.Export("nomad_id", pulumi.ToOutput(nomad_consul_token_id))
		ctx.Export("nomad_token", pulumi.ToOutput(nomad_consul_token_secret))

		// ctx.Export("cleint", client.Name)

		return nil
	})
}
