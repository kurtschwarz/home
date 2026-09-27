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

// stash is a self-hosted media organizer. It runs on nas-01 so it can scan
// the mergerfs storage pool mounted at /mnt/storage (see ansible/roles/nas):
// the pod is pinned via nodeSelector and the media pool is exposed through a
// local PersistentVolume, mounted read-only at /data since stash only scans
// and plays from it (its own state lives under /root/.stash).
const (
	nodeName        = "nas-01"
	storagePoolPath = "/mnt/storage/media"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"stash",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("stash"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionStorage exposes the nas-01 media pool to stash through the local
// volume driver: the PersistentVolume points at a path on the node's
// filesystem and is bound to nas-01 via nodeAffinity.
func provisionStorage(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	provider *kube.Provider,
) (claim *core.PersistentVolumeClaim, err error) {
	var volume *core.PersistentVolume
	if volume, err = core.NewPersistentVolume(
		ctx,
		"stash-media",
		&core.PersistentVolumeArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("stash-media"),
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
		"stash-media",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("stash-media"),
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

func provisionStash(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	media *core.PersistentVolumeClaim,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	conf := config.New(ctx, "stash")

	image := conf.Require("image")

	// /root/.stash holds the config, sqlite database, generated content,
	// metadata, cache and blobs; it must survive restarts and upgrades.
	var configClaim *core.PersistentVolumeClaim
	if configClaim, err = core.NewPersistentVolumeClaim(
		ctx,
		"stash-config",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("stash-config"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.PersistentVolumeClaimSpecArgs{
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				Resources: &core.VolumeResourceRequirementsArgs{
					Requests: &pulumi.StringMap{
						"storage": pulumi.String("20Gi"),
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
		"stash",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("stash"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Strategy: &apps.DeploymentStrategyArgs{
					Type: pulumi.String("Recreate"),
				},
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("stash"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app.kubernetes.io/name": pulumi.String("stash"),
						},
					},
					Spec: &core.PodSpecArgs{
						// pin to nas-01 so the local media volume path is valid
						NodeSelector: &pulumi.StringMap{
							"kubernetes.io/hostname": pulumi.String(nodeName),
						},
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("stash"),
								Image:           pulumi.String(image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("TZ"),
										Value: pulumi.String("America/Toronto"),
									},
									// point the library at the mounted pool
									&core.EnvVarArgs{
										Name:  pulumi.String("STASH_STASH"),
										Value: pulumi.String("/data/"),
									},
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(9999),
									},
								},
								VolumeMounts: &core.VolumeMountArray{
									&core.VolumeMountArgs{
										Name:      pulumi.String("config"),
										MountPath: pulumi.String("/root/.stash"),
									},
									// scan/playback only; keep imports away
									// from the source pool
									&core.VolumeMountArgs{
										Name:      pulumi.String("media"),
										MountPath: pulumi.String("/data"),
										ReadOnly:  pulumi.Bool(true),
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
								Name: pulumi.String("media"),
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: media.Metadata.Name().Elem(),
									ReadOnly:  pulumi.Bool(true),
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
		"stash",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("stash"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("stash"),
				},
				Ports: &core.ServicePortArray{
					&core.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(9999),
						TargetPort: pulumi.Int(9999),
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

// provisionRoute exposes the stash UI through the traefik Gateway as
// https://stash.local.kurtainerd.io (HTTP requests are redirected to HTTPS by
// the traefik web entrypoint).
func provisionRoute(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	service *core.Service,
	provider *kube.Provider,
) (route *apiextensions.CustomResource, err error) {
	return apiextensions.NewCustomResource(
		ctx,
		"stash",
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("stash"),
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
						pulumi.String("stash.local.kurtainerd.io"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": service.Metadata.Name(),
									"port": pulumi.Int(9999),
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

		var media *core.PersistentVolumeClaim
		if media, err = provisionStorage(ctx, namespace, provider); err != nil {
			return err
		}

		var service *core.Service
		if _, service, err = provisionStash(ctx, namespace, media, provider); err != nil {
			return err
		}

		if _, err = provisionRoute(ctx, namespace, service, provider); err != nil {
			return err
		}

		return nil
	})
}
