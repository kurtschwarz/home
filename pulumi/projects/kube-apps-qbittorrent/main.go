package main

import (
	kube "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	apps "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	core "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	meta "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// qbittorrent runs on nas-01: its downloads live on the mergerfs storage pool
// mounted at /mnt/storage (see ansible/roles/nas), so the pod is pinned via
// nodeSelector and the pool is exposed through a local PersistentVolume.
const (
	nodeName        = "nas-01"
	storagePoolPath = "/mnt/storage/qbittorrent"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"qbittorrent",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("qbittorrent"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionStorage exposes the nas-01 storage pool to qbittorrent through the
// local volume driver: the PersistentVolume points at a path on the node's
// filesystem and is bound to nas-01 via nodeAffinity.
func provisionStorage(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	provider *kube.Provider,
) (claim *core.PersistentVolumeClaim, err error) {
	var volume *core.PersistentVolume
	if volume, err = core.NewPersistentVolume(
		ctx,
		"qbittorrent-downloads",
		&core.PersistentVolumeArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("qbittorrent-downloads"),
			},
			Spec: &core.PersistentVolumeSpecArgs{
				Capacity: &pulumi.StringMap{
					"storage": pulumi.String("1Ti"),
				},
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				StorageClassName:              pulumi.String("local-storage"),
				PersistentVolumeReclaimPolicy: pulumi.String("Retain"),
				VolumeMode:                    pulumi.String("Filesystem"),
				Local: &core.LocalVolumeSourceArgs{
					Path: pulumi.String(storagePoolPath),
				},
				NodeAffinity: &core.VolumeNodeAffinityArgs{
					Required: &core.NodeSelectorArgs{
						NodeSelectorTerms: &core.NodeSelectorTermArray{
							&core.NodeSelectorTermArgs{
								MatchExpressions: &core.NodeSelectorRequirementArray{
									&core.NodeSelectorRequirementArgs{
										Key:      pulumi.String("kubernetes.io/hostname"),
										Operator: pulumi.String("In"),
										Values: &pulumi.StringArray{
											pulumi.String(nodeName),
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
	); err != nil {
		return nil, err
	}

	return core.NewPersistentVolumeClaim(
		ctx,
		"qbittorrent-downloads",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent-downloads"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.PersistentVolumeClaimSpecArgs{
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				Resources: &core.VolumeResourceRequirementsArgs{
					Requests: &pulumi.StringMap{
						"storage": pulumi.String("1Ti"),
					},
				},
				StorageClassName: pulumi.String("local-storage"),
				VolumeName:       volume.Metadata.Name().Elem(),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionWireguardSecret stores the WireGuard private key (from the
// encrypted Pulumi stack config) in a Kubernetes Secret, keeping it out of
// the pod spec.
func provisionWireguardSecret(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	provider *kube.Provider,
) (secret *core.Secret, err error) {
	conf := config.New(ctx, "qbittorrent")

	return core.NewSecret(
		ctx,
		"qbittorrent-wireguard",
		&core.SecretArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent-wireguard"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"private-key": conf.RequireSecret("wireguardPrivateKey"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionQbittorrent deploys qbittorrent behind a gluetun sidecar: all pod
// egress is routed through ProtonVPN over WireGuard with a killswitch, while
// cluster/LAN traffic (including the web UI path from traefik) bypasses the
// tunnel via FIREWALL_OUTBOUND_SUBNETS. The NAT-PMP forwarded port is pushed
// into qbittorrent's listen_port by gluetun's port forwarding up command.
func provisionQbittorrent(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	downloads *core.PersistentVolumeClaim,
	wireguard *core.Secret,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	conf := config.New(ctx, "qbittorrent")

	image := conf.Require("image")
	gluetunImage := conf.Require("gluetunImage")

	var configClaim *core.PersistentVolumeClaim
	if configClaim, err = core.NewPersistentVolumeClaim(
		ctx,
		"qbittorrent-config",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent-config"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.PersistentVolumeClaimSpecArgs{
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				Resources: &core.VolumeResourceRequirementsArgs{
					Requests: &pulumi.StringMap{
						"storage": pulumi.String("1Gi"),
					},
				},
			},
		},
		pulumi.Provider(provider),
	); err != nil {
		return nil, nil, err
	}

	if deployment, err = apps.NewDeployment(
		ctx,
		"qbittorrent",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Strategy: &apps.DeploymentStrategyArgs{
					Type: pulumi.String("Recreate"),
				},
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("qbittorrent"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app.kubernetes.io/name": pulumi.String("qbittorrent"),
						},
					},
					Spec: &core.PodSpecArgs{
						// pin to nas-01 so the local volume path is valid
						NodeSelector: &pulumi.StringMap{
							"kubernetes.io/hostname": pulumi.String(nodeName),
						},
						// the mergerfs pool is root-owned; qbittorrent runs as
						// PUID/PGID 1000, so grant writes through fsGroup
						SecurityContext: &core.PodSecurityContextArgs{
							FsGroup:             pulumi.Int(1000),
							FsGroupChangePolicy: pulumi.String("OnRootMismatch"),
						},
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("gluetun"),
								Image:           pulumi.String(gluetunImage),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								SecurityContext: &core.SecurityContextArgs{
									Capabilities: &core.CapabilitiesArgs{
										Add: &pulumi.StringArray{
											pulumi.String("NET_ADMIN"),
										},
									},
								},
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("VPN_SERVICE_PROVIDER"),
										Value: pulumi.String("custom"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("VPN_TYPE"),
										Value: pulumi.String("wireguard"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WIREGUARD_ADDRESSES"),
										Value: pulumi.String(conf.Require("wireguardAddress")),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WIREGUARD_ENDPOINT_IP"),
										Value: pulumi.String(conf.Require("wireguardEndpointIp")),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WIREGUARD_ENDPOINT_PORT"),
										Value: pulumi.String(conf.Require("wireguardEndpointPort")),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WIREGUARD_PUBLIC_KEY"),
										Value: pulumi.String(conf.Require("wireguardPublicKey")),
									},
									&core.EnvVarArgs{
										Name: pulumi.String("WIREGUARD_PRIVATE_KEY"),
										ValueFrom: &core.EnvVarSourceArgs{
											SecretKeyRef: &core.SecretKeySelectorArgs{
												Name: wireguard.Metadata.Name().Elem(),
												Key:  pulumi.String("private-key"),
											},
										},
									},
									// LAN + pod/service/external CIDRs bypass the
									// tunnel so the web UI stays reachable; all
									// other egress goes via the VPN (killswitch
									// blocks it if the tunnel drops). Gluetun's
									// inbound firewall drops non-tunnel traffic by
									// default, so the web UI port must be allowed
									// in on eth0 for traefik, kubelet probes and
									// port-forwards to reach the pod.
									&core.EnvVarArgs{
										Name:  pulumi.String("FIREWALL_OUTBOUND_SUBNETS"),
										Value: pulumi.String("10.32.0.0/14,10.33.0.0/16,10.34.0.0/16,10.35.0.0/16"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("FIREWALL_INPUT_PORTS"),
										Value: pulumi.String("8080"),
									},
									// ProtonVPN NAT-PMP port forwarding; when the
									// port is (re)assigned, push it into
									// qbittorrent's listen_port over the WebUI API
									// on localhost (shared pod network namespace,
									// auth bypassed via WebUI\LocalHostAuth=false
									// in the persisted qBittorrent.conf) and bind
									// peer connections to the tunnel interface.
									// The retry loop covers qbittorrent still
									// starting when the hook first fires.
									// ref: https://github.com/qdm12/gluetun-wiki/blob/main/setup/advanced/vpn-port-forwarding.md
									&core.EnvVarArgs{
										Name:  pulumi.String("VPN_PORT_FORWARDING"),
										Value: pulumi.String("on"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("VPN_PORT_FORWARDING_PROVIDER"),
										Value: pulumi.String("protonvpn"),
									},
									&core.EnvVarArgs{
										Name: pulumi.String("VPN_PORT_FORWARDING_UP_COMMAND"),
										Value: pulumi.String(`/bin/sh -c 'for i in 1 2 3 4 5 6 7 8 9 10 11 12; do wget -qO- --post-data "json={\"listen_port\":{{PORT}},\"current_network_interface\":\"{{VPN_INTERFACE}}\",\"random_port\":false}" http://127.0.0.1:8080/api/v2/app/setPreferences && exit 0; sleep 5; done; exit 1'`),
									},
									// on tunnel teardown, stop announcing the
									// stale port and bind to loopback so
									// qbittorrent cannot leak on eth0 while the
									// VPN reconnects
									&core.EnvVarArgs{
										Name:  pulumi.String("VPN_PORT_FORWARDING_DOWN_COMMAND"),
										Value: pulumi.String(`/bin/sh -c 'wget -qO- --post-data "json={\"listen_port\":0,\"current_network_interface\":\"lo\"}" http://127.0.0.1:8080/api/v2/app/setPreferences'`),
									},
								},
							},
							&core.ContainerArgs{
								Name:            pulumi.String("qbittorrent"),
								Image:           pulumi.String(image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("PUID"),
										Value: pulumi.String("1000"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("PGID"),
										Value: pulumi.String("1000"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("TZ"),
										Value: pulumi.String("America/Toronto"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WEBUI_PORT"),
										Value: pulumi.String("8080"),
									},
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
									},
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(6881),
										Protocol:      pulumi.String("TCP"),
									},
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(6881),
										Protocol:      pulumi.String("UDP"),
									},
								},
								VolumeMounts: &core.VolumeMountArray{
									&core.VolumeMountArgs{
										Name:      pulumi.String("config"),
										MountPath: pulumi.String("/config"),
									},
									&core.VolumeMountArgs{
										Name:      pulumi.String("downloads"),
										MountPath: pulumi.String("/downloads"),
									},
									&core.VolumeMountArgs{
										Name:      pulumi.String("downloads"),
										MountPath: pulumi.String("/incomplete-downloads"),
										SubPath:   pulumi.String("incomplete"),
									},
								},
							},
						},
						Volumes: &core.VolumeArray{
							core.VolumeArgs{
								Name: pulumi.String("config"),
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: configClaim.Metadata.Name().Elem(),
								},
							},
							core.VolumeArgs{
								Name: pulumi.String("downloads"),
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: downloads.Metadata.Name().Elem(),
								},
							},
						},
					},
				},
			},
		},
		pulumi.Provider(provider),
	); err != nil {
		return nil, nil, err
	}

	if service, err = core.NewService(
		ctx,
		"qbittorrent",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("qbittorrent"),
				},
				Ports: &core.ServicePortArray{
					&core.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(8080),
						TargetPort: pulumi.Int(8080),
					},
				},
			},
		},
		pulumi.Provider(provider),
	); err != nil {
		return nil, nil, err
	}

	return deployment, service, nil
}

// provisionRoute exposes the qbittorrent UI through the traefik Gateway as
// https://qbittorrent.local.kurtainerd.io (HTTP requests are redirected to
// HTTPS by the traefik web entrypoint).
func provisionRoute(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	service *core.Service,
	provider *kube.Provider,
) (route *apiextensions.CustomResource, err error) {
	return apiextensions.NewCustomResource(
		ctx,
		"qbittorrent",
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("qbittorrent"),
				Namespace: namespace.Metadata.Name(),
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
						pulumi.String("qbittorrent.local.kurtainerd.io"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": service.Metadata.Name(),
									"port": pulumi.Int(8080),
								},
							},
						},
					},
				},
			},
		},
		pulumi.Provider(provider),
	)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		var provider *kube.Provider
		if provider, err = kube.NewProvider(
			ctx,
			"homelab",
			&kube.ProviderArgs{
				Kubeconfig: pulumi.String("/home/pulumi/.kube/config"),
				Context:    pulumi.String("homelab"),
			},
		); err != nil {
			return err
		}

		var namespace *core.Namespace
		if namespace, err = provisionNamespace(ctx, provider); err != nil {
			return err
		}

		var downloads *core.PersistentVolumeClaim
		if downloads, err = provisionStorage(ctx, namespace, provider); err != nil {
			return err
		}

		var wireguard *core.Secret
		if wireguard, err = provisionWireguardSecret(ctx, namespace, provider); err != nil {
			return err
		}

		var service *core.Service
		if _, service, err = provisionQbittorrent(ctx, namespace, downloads, wireguard, provider); err != nil {
			return err
		}

		if _, err = provisionRoute(ctx, namespace, service, provider); err != nil {
			return err
		}

		return nil
	})
}
