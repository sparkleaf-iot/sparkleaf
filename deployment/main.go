package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"os"
)

func readFileOrPanic(path string, ctx *pulumi.Context) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}

	// return pulumi.String(string(data))
	return string(data)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the configuration values from the appropriate yaml.
		gcpConf := config.New(ctx, "gc")
		machineImage := gcpConf.Require("machine_image")

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

		// // Create a new subnet within the VPC network.
		// subnet, err := compute.NewSubnetwork(ctx, "nomad-subnet", &compute.SubnetworkArgs{
		// 	Region:                pulumi.String("europe-central2"),
		// 	IpCidrRange:           pulumi.String("10.0.0.0/8"),
		// 	Network:               network.ID(),
		// 	PrivateIpGoogleAccess: pulumi.Bool(true),
		// })
		// if err != nil {
		// 	return err
		// }

		// static, err := compute.NewAddress(ctx, "static", &compute.AddressArgs{
		// 	AddressType: pulumi.String("INTERNAL"),
		// 	Region:      pulumi.String("europe-central2"),
		// 	Subnetwork:  subnet.ID(),
		// })
		// if err != nil {
		// 	return err
		// }
		// instanceIp, err := compute.NewAddress(ctx, "static", &compute.AddressArgs{
		// 	Region: pulumi.String("europe-central2"),
		// })
		// if err != nil {
		// 	return err
		// }

		//test script
		// startupScript := `#!/bin/bash
		// echo "Hello, World!" > index.html
		// nohup python -m SimpleHTTPServer 80 &`
		serverStartupScript := readFileOrPanic("config/user-data-server.sh", ctx)
		// Create a new GCP compute instance to run the Nomad servers on.
		server, err := compute.NewInstance(ctx, "nomad-server", &compute.InstanceArgs{
			MachineType:            pulumi.String("e2-micro"),
			Zone:                   pulumi.String("europe-central2-a"),
			MetadataStartupScript:  pulumi.String(serverStartupScript),
			AllowStoppingForUpdate: pulumi.Bool(true),
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
		}, pulumi.DependsOn([]pulumi.Resource{firewall}))
		if err != nil {
			return err
		}
		// // Create a new GCP compute instance to run the Nomad cleints on.
		// client, err := compute.NewInstance(ctx, "nomad-client", &compute.InstanceArgs{
		// 	MachineType: pulumi.String("e2-standard-2"),
		// 	Zone:        pulumi.String("europe-central2-a"),
		// 	BootDisk: &compute.InstanceBootDiskArgs{
		// 		InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
		// 			Image: pulumi.String(machineImage),
		// 		},
		// 	},
		// 	NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
		// 		&compute.InstanceNetworkInterfaceArgs{
		// 			Network: network.SelfLink,
		// 			AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
		// 				&compute.InstanceNetworkInterfaceAccessConfigArgs{
		// 					NatIp: instanceIp.Address,
		// 				},
		// 			},
		// 			Subnetwork: subnet.SelfLink,
		// 		},
		// 	},
		// })
		// if err != nil {
		// 	return err
		// }

		// // Create a new Nomad provider that will connect to the server instance.
		// provider, err := nomad.NewProvider(ctx, "nomad-provider", &nomad.ProviderArgs{
		// 	Address: instanceIp.Address,
		// })

		// if err != nil {
		// 	return err
		// }

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
		ctx.Export("instanceIP", server.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
		// ctx.Export("cleint", client.Name)

		return nil
	})
}
