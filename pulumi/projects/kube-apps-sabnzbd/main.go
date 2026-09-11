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

// sabnzbd runs on nas-01: its downloads live on the mergerfs storage pool
// mounted at /mnt/storage (see ansible/roles/nas), so the pod is pinned via
// nodeSelector and the pool is exposed through a local PersistentVolume.
const (
	nodeName        = "nas-01"
	storagePoolPath = "/mnt/storage/sabnzbd"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"sabnzbd",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("sabnzbd"),
			},
		},
		pulumi.Provider(provider),
	)
}

// provisionStorage exposes the nas-01 storage pool to sabnzbd through the
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
		"sabnzbd-downloads",
		&core.PersistentVolumeArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("sabnzbd-downloads"),
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
		"sabnzbd-downloads",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("sabnzbd-downloads"),
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

func provisionSabnzbd(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	downloads *core.PersistentVolumeClaim,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	conf := config.New(ctx, "sabnzbd")

	image := conf.Require("image")

	var configClaim *core.PersistentVolumeClaim
	if configClaim, err = core.NewPersistentVolumeClaim(
		ctx,
		"sabnzbd-config",
		&core.PersistentVolumeClaimArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("sabnzbd-config"),
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
		"sabnzbd",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("sabnzbd"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Strategy: &apps.DeploymentStrategyArgs{
					Type: pulumi.String("Recreate"),
				},
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("sabnzbd"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app.kubernetes.io/name": pulumi.String("sabnzbd"),
						},
					},
					Spec: &core.PodSpecArgs{
						// pin to nas-01 so the local volume path is valid
						NodeSelector: &pulumi.StringMap{
							"kubernetes.io/hostname": pulumi.String(nodeName),
						},
						// the mergerfs pool is root-owned; sabnzbd runs as
						// PUID/PGID 1000, so grant writes through fsGroup
						SecurityContext: &core.PodSecurityContextArgs{
							FsGroup:             pulumi.Int(1000),
							FsGroupChangePolicy: pulumi.String("OnRootMismatch"),
						},
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("sabnzbd"),
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
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
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
		"sabnzbd",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("sabnzbd"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("sabnzbd"),
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

// provisionRoute exposes the sabnzbd UI through the traefik Gateway as
// https://sabnzbd.local.kurtainerd.io (HTTP requests are redirected to HTTPS
// by the traefik web entrypoint).
func provisionRoute(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	service *core.Service,
	provider *kube.Provider,
) (route *apiextensions.CustomResource, err error) {
	return apiextensions.NewCustomResource(
		ctx,
		"sabnzbd",
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("sabnzbd"),
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
						pulumi.String("sabnzbd.local.kurtainerd.io"),
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

		var service *core.Service
		if _, service, err = provisionSabnzbd(ctx, namespace, downloads, provider); err != nil {
			return err
		}

		if _, err = provisionRoute(ctx, namespace, service, provider); err != nil {
			return err
		}

		return nil
	})
}
