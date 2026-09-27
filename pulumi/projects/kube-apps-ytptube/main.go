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

// ytptube runs on nas-01: its downloads live on the mergerfs storage pool
// mounted at /mnt/storage (see ansible/roles/nas), so the pod is pinned via
// nodeSelector and the pool is exposed through a local PersistentVolume.
const (
	nodeName        = "nas-01"
	storagePoolPath = "/mnt/storage/ytptube"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"ytptube",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("ytptube"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionStorage exposes the nas-01 storage pool to ytptube through the
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
		"ytptube-downloads",
		&core.PersistentVolumeArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("ytptube-downloads"),
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
		"ytptube-downloads",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("ytptube-downloads"),
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

func provisionYtptube(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	downloads *core.PersistentVolumeClaim,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	conf := config.New(ctx, "ytptube")

	image := conf.Require("image")

	var configClaim *core.PersistentVolumeClaim
	if configClaim, err = core.NewPersistentVolumeClaim(
		ctx,
		"ytptube-config",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("ytptube-config"),
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
		"ytptube",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("ytptube"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Strategy: &apps.DeploymentStrategyArgs{
					Type: pulumi.String("Recreate"),
				},
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("ytptube"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app.kubernetes.io/name": pulumi.String("ytptube"),
						},
					},
					Spec: &core.PodSpecArgs{
						// pin to nas-01 so the local volume path is valid
						NodeSelector: &pulumi.StringMap{
							"kubernetes.io/hostname": pulumi.String(nodeName),
						},
						// ytptube runs as the container's user/group rather
						// than PUID/PGID env vars; the mergerfs pool is
						// root-owned, so pin the runtime user and grant
						// writes through fsGroup
						SecurityContext: &core.PodSecurityContextArgs{
							RunAsUser:           pulumi.Int(1000),
							RunAsGroup:          pulumi.Int(1000),
							FsGroup:             pulumi.Int(1000),
							FsGroupChangePolicy: pulumi.String("OnRootMismatch"),
						},
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("ytptube"),
								Image:           pulumi.String(image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("TZ"),
										Value: pulumi.String("America/Toronto"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("YTP_DOWNLOAD_PATH"),
										Value: pulumi.String("/downloads/files"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("YTP_TEMP_PATH"),
										Value: pulumi.String("/downloads/tmp"),
									},
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(8081),
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
		"ytptube",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("ytptube"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("ytptube"),
				},
				Ports: &core.ServicePortArray{
					&core.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(8081),
						TargetPort: pulumi.Int(8081),
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

// provisionRoute exposes the ytptube UI through the traefik Gateway as
// https://ytptube.local.kurtainerd.io (HTTP requests are redirected to
// HTTPS by the traefik web entrypoint).
func provisionRoute(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	service *core.Service,
	provider *kube.Provider,
) (route *apiextensions.CustomResource, err error) {
	return apiextensions.NewCustomResource(
		ctx,
		"ytptube",
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("ytptube"),
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
						pulumi.String("ytptube.local.kurtainerd.io"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": service.Metadata.Name(),
									"port": pulumi.Int(8081),
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

		var service *core.Service
		if _, service, err = provisionYtptube(ctx, namespace, downloads, provider); err != nil {
			return err
		}

		if _, err = provisionRoute(ctx, namespace, service, provider); err != nil {
			return err
		}

		return nil
	})
}
