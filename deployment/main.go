package main

import (
	"os"

	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-nomad/sdk/go/nomad"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func readFileOrPanic(path string, ctx *pulumi.Context) pulumi.StringInput {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}

	return pulumi.String(string(data))
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the configuration values from the appropriate yaml.
		conf := config.New(ctx, "")
		name := conf.Require("name")

		// Create a new VPC network for the Nomad server.
		network, err := compute.NewNetwork(ctx, "nomad-network", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Create a new subnet within the VPC network.
		subnet, err := compute.NewSubnetwork(ctx, "nomad-subnet", &compute.SubnetworkArgs{
			Region:                pulumi.String("europe-central2"),
			IpCidrRange:           pulumi.String("10.0.0.0/8"),
			Network:               network.ID(),
			PrivateIpGoogleAccess: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// static, err := compute.NewAddress(ctx, "static", &compute.AddressArgs{
		// 	AddressType: pulumi.String("INTERNAL"),
		// 	Region:      pulumi.String("europe-central2"),
		// 	Subnetwork:  subnet.ID(),
		// })
		// if err != nil {
		// 	return err
		// }
		instanceIp, err := compute.NewAddress(ctx, "static", &compute.AddressArgs{
			Region: pulumi.String("europe-central2"),
		})
		if err != nil {
			return err
		}

		// Create a new GCP compute instance to run the Nomad server on.
		server, err := compute.NewInstance(ctx, "nomad-server", &compute.InstanceArgs{
			MachineType: pulumi.String("e2-standard-2"),
			Zone:        pulumi.String("europe-central2-a"),
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String("ubuntu-os-cloud/ubuntu-2004-lts"),
				},
			},
			NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network: network.SelfLink,
					AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
						&compute.InstanceNetworkInterfaceAccessConfigArgs{
							NatIp: instanceIp.Address,
						},
					},
					Subnetwork: subnet.SelfLink,
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Log.Info(instanceIp.Address, nil)

		// Create a firewall rule to allow traffic to the Nomad server.
		_, err = compute.NewFirewall(ctx, "nomad-firewall", &compute.FirewallArgs{
			Network: network.SelfLink,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("22"),
						pulumi.String("4646"),
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
		// Create a new Nomad provider that will connect to the server instance.
		provider, err := nomad.NewProvider(ctx, "nomad-provider", &nomad.ProviderArgs{
			Address: instanceIp.Address + pulumi.StringOutput(":4646"),
		})

		// if err != nil {
		// 	return err
		// }

		traefikJob, err := nomad.NewJob(ctx, "traefik-cluster", &nomad.JobArgs{
			Jobspec: readFileOrPanic("jobs/traefik.nomad.hcl", ctx),
		}, pulumi.Provider(provider))

		// if err != nil {
		// 	return err
		// }

		// influxJob, err := nomad.NewJob(ctx, "influx-cluster", &nomad.JobArgs{
		// 	Jobspec: readFileOrPanic("jobs/influx.nomad.hcl", ctx),
		// }, pulumi.Provider(provider))

		// if err != nil {
		// 	return err
		// }

		ctx.Export("traefikJob", traefikJob.ID())
		// ctx.Export("influxJob", influxJob.ID())
		ctx.Export("server", server.Name)

		return nil
	})
}
