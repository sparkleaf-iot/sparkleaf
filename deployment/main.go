package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-nomad/sdk/go/nomad"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the configuration values from the appropriate yaml.
		gcpConf := config.New(ctx, "gc")
		generalConf := config.New(ctx, "")
		instanceCount, err := generalConf.TryInt("instance_count")
		if err != nil {
			instanceCount = 3
		}
		machineImage := gcpConf.Require("machine_image")
		// Create a bootstrap token for Consul and Nomad
		nomad_consul_token_id := pulumi.ToSecret(createToken())
		nomad_consul_token_secret := pulumi.ToSecret(createToken())

		network, err := createNetwork(ctx)
		if err != nil {
			return err
		}
		firewall, err := createFirewall(ctx, network)
		if err != nil {
			return err
		}
		internalFirewall, err := createInternalFirewall(ctx, network)
		if err != nil {
			return err
		}
		serviceAccount, err := serviceaccount.NewAccount(ctx, "serviceAccount", &serviceaccount.AccountArgs{
			AccountId:   pulumi.String("server-account"),
			DisplayName: pulumi.String("Vm server service account"),
		})

		if err != nil {
			return err
		}
		_, err = projects.NewIAMMember(ctx, "disk-iam", &projects.IAMMemberArgs{
			Project: pulumi.String("sparkleaf"),
			Role:    pulumi.String("projects/sparkleaf/roles/diskIo"),
			Member:  serviceAccount.Member,
		})
		if err != nil {
			return err
		}

		serviceAccountKey, err := serviceaccount.NewKey(ctx, "instanceKey", &serviceaccount.KeyArgs{
			ServiceAccountId: serviceAccount.Name,
		})
		if err != nil {
			return err
		}
		dnsZone := dns.LookupManagedZoneOutput(ctx, dns.LookupManagedZoneOutputArgs{
			Name: pulumi.String("sparkleaf-main"),
		}, nil)
		if err != nil {
			return err
		}

		influxDisk, err := compute.NewDisk(ctx, "influxdisk", &compute.DiskArgs{
			Size: pulumi.Int(10),
			Type: pulumi.String("pd-standard"),
			Zone: pulumi.String("europe-central2-b"),
		}, pulumi.Protect(false))
		if err != nil {
			return err
		}
		serverScriptFile := readFileOrPanic("config/server.sh", ctx)
		serverStartupScript := pulumi.All(nomad_consul_token_id, nomad_consul_token_secret).ApplyT(
			func(args []interface{}) string {

				script := injectToken(args[0].(string), "nomad_consul_token_id", serverScriptFile, 1)
				script = injectToken(args[1].(string), "nomad_consul_token_secret", script, 2)
				return script
			})

		var server []*compute.Instance
		for i := 0; i < instanceCount; i++ {
			time.Sleep(time.Second * 2)
			serverScript := serverStartupScript.ApplyT(func(script string) string {
				return injectToken(strconv.Itoa(i), "INSTANCE_NUMBER_PLACEHOLDER", script, 1)
			})
			__res, err := compute.NewInstance(ctx, fmt.Sprintf("server-%d", i), &compute.InstanceArgs{
				MachineType:           pulumi.String("e2-micro"),
				Zone:                  pulumi.String("europe-central2-b"),
				MetadataStartupScript: pulumi.Sprintf("%s", serverScript),
				Metadata: pulumi.StringMap{
					"access_token": pulumi.String("nil"),
				},
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
			}, pulumi.DependsOn([]pulumi.Resource{
				firewall,
				internalFirewall,
				serviceAccountKey,
			}), pulumi.IgnoreChanges([]string{"metadataStartupScript"}))
			if err != nil {
				return err
			}
			server = append(server, __res)

		}

		// Create a new GCP compute instance to run the Nomad cleints on.
		clientScriptFile := readFileOrPanic("config/client.sh", ctx)
		clientStartupScript := pulumi.All(nomad_consul_token_secret).ApplyT(
			func(args []interface{}) string {
				script := injectToken(args[0].(string), "nomad_consul_token_secret", clientScriptFile, 2)
				return script
			})

		var client []*compute.Instance
		for i := 0; i < instanceCount; i++ {
			__res, err := compute.NewInstance(ctx, fmt.Sprintf("client-%d", i), &compute.InstanceArgs{
				MachineType:            pulumi.String("e2-micro"),
				Zone:                   pulumi.String("europe-central2-b"),
				MetadataStartupScript:  pulumi.Sprintf("%s", clientStartupScript),
				AllowStoppingForUpdate: pulumi.Bool(true),
				Tags:                   pulumi.StringArray{pulumi.String("auto-join"), pulumi.String("nomad-clients")},
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
			}, pulumi.DependsOn([]pulumi.Resource{firewall, internalFirewall}), pulumi.IgnoreChanges([]string{"metadataStartupScript"}))
			if err != nil {
				return err
			}
			client = append(client, __res)

		}

		if err != nil {
			return err
		}

		clientIpArray := pulumi.StringArray{}
		for key := range client {
			ip := client[key].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp().ApplyT(func(ip *string) string {
				return *ip
			}).(pulumi.StringOutput)

			clientIpArray = append(clientIpArray, ip)
		}

		_, err = dns.NewRecordSet(ctx, "clientRecordset", &dns.RecordSetArgs{
			Name:        pulumi.String("sparkleaf.emilsallem.com."),
			Type:        pulumi.String("A"),
			Ttl:         pulumi.Int(300),
			ManagedZone: dnsZone.Name().ApplyT(func(name string) string { return name }).(pulumi.StringOutput),
			Rrdatas:     clientIpArray,
		})
		_, err = dns.NewRecordSet(ctx, "influxRecordSet", &dns.RecordSetArgs{
			Name:        pulumi.String("influx.emilsallem.com."),
			Type:        pulumi.String("A"),
			Ttl:         pulumi.Int(300),
			ManagedZone: dnsZone.Name().ApplyT(func(name string) string { return name }).(pulumi.StringOutput),
			Rrdatas:     clientIpArray,
		})

		if err != nil {
			return err
		}

		natIp := server[0].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp()

		url := natIp.ApplyT(func(ip *string) string {
			return "http://" + *ip + ":4646"
		}).(pulumi.StringOutput)

		consulKvUrl := natIp.ApplyT(func(ip *string) string {
			return "http://" + *ip + ":8500/v1/kv/"
		}).(pulumi.StringOutput)

		accountKey := pulumi.All(consulKvUrl, serviceAccountKey.PrivateKey, nomad_consul_token_secret).ApplyT(
			func(args []interface{}) []byte {
				return setAccountKey(args[0].(string), args[1].(string), args[2].(string))
			})

		accessToken := pulumi.All(consulKvUrl, nomad_consul_token_secret).ApplyT(
			func(args []interface{}) string {
				return getAccessToken(args[0].(string), args[1].(string))
			}).(pulumi.StringOutput)

		provider, err := nomad.NewProvider(ctx, "nomad", &nomad.ProviderArgs{
			Address:     url,
			SecretId:    accessToken,
			ConsulToken: pulumi.Sprintf("%s", nomad_consul_token_secret),
		}, pulumi.DependsOn([]pulumi.Resource{server[0]}), pulumi.IgnoreChanges([]string{"secretId"}))

		if err != nil {
			return err
		}

		traefikJobSpec := nomad_consul_token_secret.ApplyT(func(token string) string {
			return injectToken(token, "nomad_consul_token_secret", readFileOrPanic("jobs/traefik.nomad.hcl", ctx), 1)
		})

		traefikJob, err := nomad.NewJob(ctx, "traefik", &nomad.JobArgs{
			Jobspec: pulumi.Sprintf("%s", traefikJobSpec),
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DependsOn([]pulumi.Resource{provider}))

		if err != nil {
			return err
		}

		csiControllerJob, err := nomad.NewJob(ctx, "csi-controller", &nomad.JobArgs{
			Jobspec: pulumi.String(readFileOrPanic("jobs/csi-controller.nomad.hcl", ctx)),
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DependsOn([]pulumi.Resource{provider}))

		if err != nil {
			return err
		}

		csiNodeJob, err := nomad.NewJob(ctx, "csi-node", &nomad.JobArgs{
			Jobspec: pulumi.String(readFileOrPanic("jobs/csi-node.nomad.hcl", ctx)),
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn([]pulumi.Resource{provider}))

		if err != nil {
			return err
		}
		csiPlugin := nomad.GetPluginOutput(ctx, nomad.GetPluginOutputArgs{
			PluginId:       pulumi.String("gcepd"),
			WaitForHealthy: pulumi.Bool(true),
		}, nil)

		_ = csiPlugin.Id().ApplyT(func(id string) (interface{}, error) {

			volume, err := nomad.NewVolume(ctx, "influxVolume", &nomad.VolumeArgs{
				Type:       pulumi.String("csi"),
				PluginId:   pulumi.String("gcepd"),
				VolumeId:   pulumi.String("influx_volume"),
				ExternalId: influxDisk.ID(),
				Capabilities: nomad.VolumeCapabilityArray{
					&nomad.VolumeCapabilityArgs{
						AccessMode:     pulumi.String("single-node-writer"),
						AttachmentMode: pulumi.String("file-system"),
					},
				},
				MountOptions: &nomad.VolumeMountOptionsArgs{
					FsType: pulumi.String("ext4"),
				},
			}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{provider, csiControllerJob, csiNodeJob}))
			if err != nil {
				return nil, err
			}
			return volume, err
		})

		influxJob, err := nomad.NewJob(ctx, "influx-cluster", &nomad.JobArgs{
			Jobspec: pulumi.String(readFileOrPanic("jobs/influx.nomad.hcl", ctx)),
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{csiControllerJob, csiNodeJob}))

		if err != nil {
			return err
		}

		ctx.Export("nomad_job_token", accessToken)
		ctx.Export("influxJob", influxJob.ID())
		ctx.Export("traefikJob", traefikJob.ID())
		ctx.Export("csiControllerJob", csiControllerJob.ID())
		for key := range server {
			ctx.Export("server"+strconv.Itoa(key), server[key].Name)
			ctx.Export("serverIP"+strconv.Itoa(key), server[key].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
			ctx.Export("client"+strconv.Itoa(key), client[key].Name)
			ctx.Export("clientIP"+strconv.Itoa(key), client[key].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
		}
		ctx.Export("nomad_id", pulumi.ToOutput(nomad_consul_token_id))
		ctx.Export("consul_token", pulumi.ToOutput(nomad_consul_token_secret))
		ctx.Export("account_key", accountKey)

		return nil
	})
}
