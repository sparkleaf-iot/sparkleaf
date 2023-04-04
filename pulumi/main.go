package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	nomad "github.com/pulumi/pulumi-nomad/sdk/v3/go/nomad"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create a new GCP network for the Nomad cluster.
		network, err := compute.NewNetwork(ctx, "nomad-network", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Create a new GCP subnet for the Nomad cluster.
		subnet, err := compute.NewSubnetwork(ctx, "nomad-subnet", &compute.SubnetworkArgs{
			Network:       network.ID(),
			IpCidrRange:   pulumi.String("10.0.0.0/24"),
			Region:        pulumi.String("us-central1"),
			PrivateIpGoogleAccess: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
			// Create a new Nomad cluster.
			cluster, err := nomad.NewCluster(ctx, "nomad-cluster", &nomad.ClusterArgs{
				Datacenters: pulumi.StringArray{"dc1"},
				Region:      pulumi.String("us-central1"),
				SubnetIds:   pulumi.StringArray{subnet.ID()},
				Node: &nomad.ClusterNodeArgs{
					Count:          pulumi.Int(3),
					MachineType:    pulumi.String("n1-standard-2"),
					NetworkProfile: pulumi.String("custom"),
				},
			})
			if err != nil {
				return err
			}

		

		// Export the Nomad cluster address.
		ctx.Export("nomad-address", cluster.HttpAddress)

		return nil
	})
}
