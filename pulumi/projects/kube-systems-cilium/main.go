package main

import (
	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	meta "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func buildChartValuesMap(m map[string]interface{}) pulumi.Map {
	result := pulumi.Map{}
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = buildChartValuesMap(val)
		case string:
			result[k] = pulumi.String(val)
		case bool:
			result[k] = pulumi.Bool(val)
		case float64:
			result[k] = pulumi.Float64(val)
		case int:
			result[k] = pulumi.Int(val)
		default:
			result[k] = pulumi.Sprintf("%v", val)
		}
	}

	return result
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		provider, err := kubernetes.NewProvider(ctx, "homelab", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String("/home/pulumi/.kube/config"),
			Context:    pulumi.String("homelab"),
		})

		if err != nil {
			return err
		}

		conf := config.New(ctx, "cilium")

		chartValues := map[string]interface{}{}
		conf.RequireObject("chartValues", &chartValues)

		chart, err := helm.NewChart(
			ctx,
			"cilium",
			&helm.ChartArgs{
				Chart:     pulumi.String("cilium"),
				Version:   pulumi.String(conf.Require("chartVersion")),
				Namespace: pulumi.String("kube-system"),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://helm.cilium.io"),
				},
				Values: buildChartValuesMap(chartValues),
			},
			pulumi.Provider(provider),
		)

		if err != nil {
			return err
		}

		// LB-IPAM: hand LoadBalancer services an address out of the external
		// CIDR. Without a pool, `type: LoadBalancer` services stay pending.
		// Advertised to the network over BGP (see below).
		if _, err = apiextensions.NewCustomResource(
			ctx,
			"loadbalancer-ip-pool",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cilium.io/v2alpha1"),
				Kind:       pulumi.String("CiliumLoadBalancerIPPool"),
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("homelab"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": pulumi.Map{
						"blocks": pulumi.Array{
							pulumi.Map{
								"cidr": pulumi.String(conf.Require("loadBalancerCIDR")),
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart}),
		); err != nil {
			return err
		}

		// BGP: peer the on-prem nodes with the UDM-SE and advertise the
		// LoadBalancer service IPs to it, so the router installs real /32
		// routes instead of relying on L2/ARP within the VLAN.
		peerConfig, err := apiextensions.NewCustomResource(
			ctx,
			"bgp-peer-config",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cilium.io/v2"),
				Kind:       pulumi.String("CiliumBGPPeerConfig"),
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("udm-se"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": pulumi.Map{
						"families": pulumi.Array{
							pulumi.Map{
								"afi":  pulumi.String("ipv4"),
								"safi": pulumi.String("unicast"),
								"advertisements": pulumi.Map{
									"matchLabels": pulumi.Map{
										"advertise": pulumi.String("bgp"),
									},
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart}),
		)

		if err != nil {
			return err
		}

		advertisement, err := apiextensions.NewCustomResource(
			ctx,
			"bgp-advertisement",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cilium.io/v2"),
				Kind:       pulumi.String("CiliumBGPAdvertisement"),
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("services"),
					Labels: pulumi.StringMap{
						"advertise": pulumi.String("bgp"),
					},
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": pulumi.Map{
						"advertisements": pulumi.Array{
							pulumi.Map{
								"advertisementType": pulumi.String("Service"),
								"service": pulumi.Map{
									"addresses": pulumi.Array{
										pulumi.String("LoadBalancerIP"),
									},
								},
								// select every service (NotIn a value nothing uses)
								"selector": pulumi.Map{
									"matchExpressions": pulumi.Array{
										pulumi.Map{
											"key":      pulumi.String("advertise-bgp"),
											"operator": pulumi.String("NotIn"),
											"values": pulumi.Array{
												pulumi.String("never"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart}),
		)

		if err != nil {
			return err
		}

		if _, err = apiextensions.NewCustomResource(
			ctx,
			"bgp-cluster-config",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cilium.io/v2"),
				Kind:       pulumi.String("CiliumBGPClusterConfig"),
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("homelab"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": pulumi.Map{
						// on-prem nodes only; rnk-01 reaches the cluster over
						// WireGuard and does not peer with the UDM.
						"nodeSelector": pulumi.Map{
							"matchExpressions": pulumi.Array{
								pulumi.Map{
									"key":      pulumi.String("kubernetes.io/hostname"),
									"operator": pulumi.String("NotIn"),
									"values": pulumi.Array{
										pulumi.String("rnk-01"),
									},
								},
							},
						},
						"bgpInstances": pulumi.Array{
							pulumi.Map{
								"name":     pulumi.String("homelab"),
								"localASN": pulumi.Int(conf.RequireInt("bgpLocalASN")),
								"peers": pulumi.Array{
									pulumi.Map{
										"name":        pulumi.String("udm-se"),
										"peerASN":     pulumi.Int(conf.RequireInt("bgpPeerASN")),
										"peerAddress": pulumi.String(conf.Require("bgpPeerAddress")),
										"peerConfigRef": pulumi.Map{
											"name": pulumi.String("udm-se"),
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{peerConfig, advertisement}),
		); err != nil {
			return err
		}

		// expose the hubble UI through the traefik Gateway as
		// https://hubble.local.kurtainerd.io
		if _, err = apiextensions.NewCustomResource(
			ctx,
			"hubble-ui",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
				Kind:       pulumi.String("HTTPRoute"),
				Metadata: &meta.ObjectMetaArgs{
					Name:      pulumi.String("hubble-ui"),
					Namespace: pulumi.String("kube-system"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": pulumi.Map{
						"parentRefs": pulumi.Array{
							pulumi.Map{
								"name":      pulumi.String("traefik-gateway"),
								"namespace": pulumi.String("traefik"),
							},
						},
						"hostnames": pulumi.Array{
							pulumi.String("hubble.local.kurtainerd.io"),
						},
						"rules": pulumi.Array{
							pulumi.Map{
								"backendRefs": pulumi.Array{
									pulumi.Map{
										"name": pulumi.String("hubble-ui"),
										"port": pulumi.Int(80),
									},
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart}),
		); err != nil {
			return err
		}

		return nil
	})
}
