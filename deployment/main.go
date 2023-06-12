package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	//"github.com/pulumi/pulumi-nomad/sdk/go/nomad"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func readFileOrPanic(path string, ctx *pulumi.Context) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return string(data)
}

func waitForLeaderElection(url string, token string ) bool {
	client := &http.Client{}

	// Retry till leader is ready
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", url+"/v1/status/leader", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(time.Second * 15)
			log.Println("Waiting for leader election...")
		} else {
			defer resp.Body.Close()
			return true
		}
	}
	return false

}
func getAccessToken(url string, token string) string {
	// Querying the Consul KV store to fetch the access token
	client := &http.Client{}

	// Retry till Consul is ready
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", url+"nomad_user_token", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(time.Second * 15)
			log.Println("Retrying...")
		} else {
			defer resp.Body.Close()
			// Read the response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			// Extract the value from the response body
			var response []struct {
				Value string `json:"Value"`
			}
			err = json.Unmarshal(body, &response)
			if err != nil {
				log.Fatal(err)
			}

			if len(response) > 0 {
				// Resp is base64 encoded, TODO figure out better way
				value64 := response[0].Value
				value, err := base64.StdEncoding.DecodeString(value64)
				if err != nil {
					log.Fatal(err)
				}

				return string(value)
			} else {
				log.Fatal("Value not found in the response")
				return ""
			}

		}
	}

	return "Timeout error"

}

func setAccountKey(url string, key64 string, token string) []byte {
	// Querying the Consul KV store to fetch the access token
	client := &http.Client{}
	key, err := base64.StdEncoding.DecodeString(key64)
	if err != nil {
		log.Fatal(err)
	}

	// Retry till Consul is ready
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest(http.MethodPut, url+"service_account", bytes.NewBuffer([]byte(key)))
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(time.Second * 15)
			log.Println("Retrying...")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			return body
		}
	}

	return []byte("Timeout error")
}

func injectToken(token string, toBeReplaced string, script string, amount int) string {

	return strings.Replace(script, toBeReplaced, token, amount)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the configuration values from the appropriate yaml.
		gcpConf := config.New(ctx, "gc")
		generalConf := config.New(ctx, "")
		instanceCount, err := generalConf.TryInt("instance_count")
        if err != nil {
            instanceCount= 3
        }
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

		// influxDisk, err := compute.NewDisk(ctx, "influxdisk", &compute.DiskArgs{
		// 	Size: pulumi.Int(10),
		// 	Type: pulumi.String("pd-standard"),
		// 	Zone: pulumi.String("europe-central2-b"),
		// })
		// if err != nil {
		// 	return err
		// }

		serverStartupScript := readFileOrPanic("config/server.sh", ctx)
		serverStartupScript = injectToken(nomad_consul_token_id, "nomad_consul_token_id", serverStartupScript, 1)
		serverStartupScript = injectToken(nomad_consul_token_secret, "nomad_consul_token_secret", serverStartupScript, 2)

		var server []*compute.Instance
		for i:= 0; i < instanceCount; i++ {
			__res, err := compute.NewInstance(ctx, fmt.Sprintf("server-%v", i), &compute.InstanceArgs{
				MachineType:           pulumi.String("e2-micro"),
				Zone:                  pulumi.String("europe-central2-b"),
				MetadataStartupScript: pulumi.String(serverStartupScript),
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
			}, pulumi.DependsOn([]pulumi.Resource{firewall, internalFirewall, serviceAccountKey}))
			if err != nil {
				return err
			}
			server = append(server, __res)

		}

		clientStartupScript := readFileOrPanic("config/client.sh", ctx)
		clientStartupScript = injectToken(nomad_consul_token_secret, "nomad_consul_token_secret", clientStartupScript, 1)
		// Create a new GCP compute instance to run the Nomad cleints on.

		var client []*compute.Instance
		for i:= 0; i < instanceCount; i++ {
			__res, err := compute.NewInstance(ctx, fmt.Sprintf("client-%v", i), &compute.InstanceArgs{
				MachineType:            pulumi.String("e2-micro"),
				Zone:                   pulumi.String("europe-central2-b"),
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
				// handle error
			}
			client = append(client, __res)

		}

		if err != nil {
			return err
		}
		natIp := server[0].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp()

		// url := natIp.ApplyT(func(ip *string) string {
		// 	return "http://" + *ip + ":4646"
		// }).(pulumi.StringOutput)

		consulKvUrl := natIp.ApplyT(func(ip *string) string {
			return "http://" + *ip + ":8500/v1/kv/"
		}).(pulumi.StringOutput)

		accountKey := pulumi.All(consulKvUrl, serviceAccountKey.PrivateKey).ApplyT(
			func(args []interface{}) []byte {
				return setAccountKey(args[0].(string), args[1].(string), nomad_consul_token_secret)
			})
		accessToken := consulKvUrl.ApplyT(
			func(url string) string {
				return getAccessToken(url, nomad_consul_token_secret)
			}).(pulumi.StringOutput)

		// provider, err := nomad.NewProvider(ctx, "nomad", &nomad.ProviderArgs{
		// 	Address:     url,
		// 	SecretId:    accessToken,
		// 	ConsulToken: pulumi.String(nomad_consul_token_secret),
		// }, pulumi.DependsOn([]pulumi.Resource{server[0]}))

		// if err != nil {
		// 	return err
		// }
		// _ = url.ApplyT(func(url string) (interface{}, error) {
		// 	return waitForLeaderElection(url, nomad_consul_token_secret), nil
		// })

		
			// traefikJobSpec := readFileOrPanic("jobs/traefik.nomad.hcl", ctx)
			// traefikJobSpec = injectToken(nomad_consul_token_secret, "nomad_consul_token_secret", traefikJobSpec, 1)
			// traefikJob, err := nomad.NewJob(ctx, "traefik", &nomad.JobArgs{
			// 	Jobspec: pulumi.String(traefikJobSpec),
			// }, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn([]pulumi.Resource{provider}))
	
			// if err != nil {
			// 	return err
			// }
	
			// csiControllerJob, err := nomad.NewJob(ctx, "csi-controller", &nomad.JobArgs{
			// 	Jobspec: pulumi.String(readFileOrPanic("jobs/csi-controller.nomad.hcl", ctx)),
			// }, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn([]pulumi.Resource{provider}))
	
			// if err != nil {
			// 	return err
			// }
	
			// csiNodeJob, err := nomad.NewJob(ctx, "csi-node", &nomad.JobArgs{
			// 	Jobspec: pulumi.String(readFileOrPanic("jobs/csi-node.nomad.hcl", ctx)),
			// }, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"jobspec"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn([]pulumi.Resource{provider}))
	
			// if err != nil {
			// 	return err
			// }
			// csiPlugin := nomad.GetPluginOutput(ctx, nomad.GetPluginOutputArgs{
			// 	PluginId:       pulumi.String("gcepd"),
			// 	WaitForHealthy: pulumi.Bool(true),
			// }, nil)
	
			// _ = csiPlugin.Id().ApplyT(func(id string) (interface{}, error) {
	
			// 	volume, err := nomad.NewVolume(ctx, "influxVolume", &nomad.VolumeArgs{
			// 		Type:       pulumi.String("csi"),
			// 		PluginId:   pulumi.String("gcepd"),
			// 		VolumeId:   pulumi.String("influx_volume"),
			// 		ExternalId: influxDisk.ID(),
			// 		Capabilities: nomad.VolumeCapabilityArray{
			// 			&nomad.VolumeCapabilityArgs{
			// 				AccessMode:     pulumi.String("single-node-writer"),
			// 				AttachmentMode: pulumi.String("file-system"),
			// 			},
			// 		},
			// 		MountOptions: &nomad.VolumeMountOptionsArgs{
			// 			FsType: pulumi.String("ext4"),
			// 		},
			// 	}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{provider, csiControllerJob, csiNodeJob}))
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// 	return volume, err
			// })
	
			// influxJob, err := nomad.NewJob(ctx, "influx-cluster", &nomad.JobArgs{
			// 	Jobspec: pulumi.String(readFileOrPanic("jobs/influx.nomad.hcl", ctx)),
			// }, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{csiControllerJob, csiNodeJob}))
	
			// if err != nil {
			// 	return err
			// }
	
		
		

		
		ctx.Export("nomad_job_token", accessToken)
		// ctx.Export("influxJob", influxJob.ID())
		//ctx.Export("traefikJob", traefikJob.ID())
		//ctx.Export("csiControllerJob", csiControllerJob.ID())
		for key, _ := range server{
			ctx.Export("server" + strconv.Itoa(key), server[key].Name)
			ctx.Export("serverIP" + strconv.Itoa(key), server[key].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
			ctx.Export("client" + strconv.Itoa(key), client[key].Name)
			ctx.Export("clientIP" + strconv.Itoa(key), client[key].NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp())
		}
		ctx.Export("nomad_id", pulumi.ToOutput(nomad_consul_token_id))
		ctx.Export("consul_token", pulumi.ToOutput(nomad_consul_token_secret))
		ctx.Export("account_key", accountKey)

		return nil
	})
}
