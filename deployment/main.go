package main

import (
	"os"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-nomad/sdk/go/nomad"
)

func readFileOrPanic(path string) pulumi.StringInput {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return pulumi.String(string(data))
}


func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		

		// Create a new VPC network for the Nomad server.
	network, err := compute.NewNetwork(ctx, "nomad-network", &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}

	// Create a new subnet within the VPC network.
	subnet, err := compute.NewSubnetwork(ctx, "nomad-subnet", &compute.SubnetworkArgs{
		Region:          pulumi.String("us-central1"),
		IpCidrRange:     pulumi.String("10.0.1.0/24"),
		Network:         network.SelfLink,
		PrivateIpGoogleAccess: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}

	static, err := compute.NewAddress(ctx, "static", &compute.AddressArgs{
		Region:      pulumi.String("us-central1"),


})
	if err != nil {
		return err
	}

		// Create a new GCP compute instance to run the Nomad server on.
		server, err := compute.NewInstance(ctx, "nomad-server", &compute.InstanceArgs{
			MachineType: pulumi.String("e2-standard-2"),
			Zone:        pulumi.String("us-central1-a"),
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
						NatIp: static.Address,
					},
				},
				Subnetwork: subnet.SelfLink,
			
				},
			},
		})
		if err != nil {
			return err
		}

		
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
			Address: static.Address,
        })

        if err != nil {
            return err
        }


		job, err := nomad.NewJob(ctx, "nomad-cluster", &nomad.JobArgs{
            Jobspec: readFileOrPanic("jobs/traefik.nomad.hcl"),
        }, pulumi.Provider(provider))

		if err != nil {
            return err
        }

		
		ctx.Export("job", job.ID())
		ctx.Export("server", server.Name)

		return nil
	})
}
