package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createNetwork(ctx *pulumi.Context) (*compute.Network, error) {

	// Create a new VPC network for the Nomad server.
	network, err := compute.NewNetwork(ctx, "nomad-network", &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return network, nil

}

func createFirewall(ctx *pulumi.Context, network *compute.Network) (*compute.Firewall, error) {

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
					pulumi.String("8080"),
					pulumi.String("8081"),
					pulumi.String("80"),
					pulumi.String("8086"),
				},
			},
		},
		SourceRanges: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
		},
	})
	if err != nil {
		return nil, err
	}

	return firewall, nil

}

func createInternalFirewall(ctx *pulumi.Context, network *compute.Network) (*compute.Firewall, error) {

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
		return nil, err
	}
	return internalFirewall, nil
}
