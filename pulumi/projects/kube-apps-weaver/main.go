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

// weaver runs on nas-01: its downloads live on the mergerfs storage pool
// mounted at /mnt/storage (see ansible/roles/nas), so the pod is pinned via
// nodeSelector and the pool is exposed through a local PersistentVolume.
const (
	nodeName        = "nas-01"
	storagePoolPath = "/mnt/storage/weaver"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"weaver",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("weaver"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionStorage exposes the nas-01 storage pool to weaver through the
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
		"weaver-downloads",
		&core.PersistentVolumeArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("weaver-downloads"),
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
		"weaver-downloads",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("weaver-downloads"),
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

func provisionWeaver(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	downloads *core.PersistentVolumeClaim,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	conf := config.New(ctx, "weaver")

	// first-run login credentials, read only until a login exists
	// (change the password later in Settings -> Security)
	image := conf.Require("image")
	bootstrapUsername := conf.Require("bootstrapUsername")
	bootstrapPassword := conf.RequireSecret("bootstrapPassword")

	var configClaim *core.PersistentVolumeClaim
	if configClaim, err = core.NewPersistentVolumeClaim(
		ctx,
		"weaver-config",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("weaver-config"),
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
		"weaver",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("weaver"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Strategy: &apps.DeploymentStrategyArgs{
					Type: pulumi.String("Recreate"),
				},
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("weaver"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app.kubernetes.io/name": pulumi.String("weaver"),
						},
					},
					Spec: &core.PodSpecArgs{
						// pin to nas-01 so the local volume path is valid
						NodeSelector: &pulumi.StringMap{
							"kubernetes.io/hostname": pulumi.String(nodeName),
						},
						// the mergerfs pool is root-owned; weaver runs as
						// PUID/PGID 1000, so grant writes through fsGroup
						SecurityContext: &core.PodSecurityContextArgs{
							FsGroup:             pulumi.Int(1000),
							FsGroupChangePolicy: pulumi.String("OnRootMismatch"),
						},
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("weaver"),
								Image:           pulumi.String(image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("WEAVER_BOOTSTRAP_LOGIN_USERNAME"),
										Value: pulumi.String(bootstrapUsername),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WEAVER_BOOTSTRAP_LOGIN_PASSWORD"),
										Value: bootstrapPassword,
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WEAVER_HTTP_ALLOWED_HOSTS"),
										Value: pulumi.String("weaver.local.kurtainerd.io"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WEAVER_INTERMEDIATE_DIR"),
										Value: pulumi.String("/downloads/intermediate"),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("WEAVER_COMPLETE_DIR"),
										Value: pulumi.String("/downloads/complete"),
									},
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(9090),
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
		"weaver",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("weaver"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("weaver"),
				},
				Ports: &core.ServicePortArray{
					&core.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(9090),
						TargetPort: pulumi.Int(9090),
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

// provisionRoute exposes the weaver UI through the traefik Gateway as
// https://weaver.local.kurtainerd.io (HTTP requests are redirected to HTTPS
// by the traefik web entrypoint).
func provisionRoute(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	service *core.Service,
	provider *kube.Provider,
) (route *apiextensions.CustomResource, err error) {
	return apiextensions.NewCustomResource(
		ctx,
		"weaver",
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("weaver"),
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
						pulumi.String("weaver.local.kurtainerd.io"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": service.Metadata.Name(),
									"port": pulumi.Int(9090),
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
		if _, service, err = provisionWeaver(ctx, namespace, downloads, provider); err != nil {
			return err
		}

		if _, err = provisionRoute(ctx, namespace, service, provider); err != nil {
			return err
		}

		return nil
	})
}
