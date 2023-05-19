package main

import (
	infra "github.com/kurtschwarz/home/packages/infrastructure"
	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func provisionSecretEnvs(ctx *pulumi.Context, config *config.Config, namespace pulumi.StringPtrInput) (*corev1.Secret, error) {
	secretEnvs, err := corev1.NewSecret(ctx, "plex-secret-envs", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("plex-secret-envs"),
			Namespace: namespace,
		},
		StringData: pulumi.StringMap{
			"PLEX_CLAIM": config.RequireSecret("claimToken"),
		},
	})

	if err != nil {
		return nil, err
	}

	return secretEnvs, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "plex")
		namespace := infra.RequireNamespace(ctx, "homelab-apps-kurtflix", ctx.Stack())

		certificate, err := infra.ProvisionCertificate(ctx, namespace, config.Require("domain"))
		if err != nil {
			return err
		}

		secretEnvs, err := provisionSecretEnvs(ctx, config, namespace)
		if err != nil {
			return err
		}

		var configVolumeClaim *corev1.PersistentVolumeClaim
		if configVolumeClaim, err = corev1.NewPersistentVolumeClaim(
			ctx,
			"plex-config-pvc",
			&corev1.PersistentVolumeClaimArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("plex-config-pvc"),
					Namespace: namespace,
					Annotations: &pulumi.StringMap{
						"pulumi.com/skipAwait": pulumi.String("true"),
					},
				},
				Spec: &corev1.PersistentVolumeClaimSpecArgs{
					StorageClassName: pulumi.String("longhorn"),
					Resources: &corev1.ResourceRequirementsArgs{
						Requests: pulumi.StringMap{
							"storage": pulumi.String("20Gi"),
						},
					},
					AccessModes: &pulumi.StringArray{
						pulumi.String("ReadWriteOnce"),
					},
				},
			},
		); err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("plex"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		deployment, err := appsv1.NewDeployment(
			ctx,
			"plex-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace,
					Labels:    deploymentLabels,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: deploymentLabels,
					},
					Replicas: pulumi.Int(config.RequireInt("replicas")),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Namespace: namespace,
							Labels:    deploymentLabels,
						},
						Spec: &corev1.PodSpecArgs{
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("plex"),
									Image: pulumi.String(config.Require("image")),
									ReadinessProbe: &corev1.ProbeArgs{
										TcpSocket: &corev1.TCPSocketActionArgs{
											Port: pulumi.Int(32400),
										},
										InitialDelaySeconds: pulumi.Int(20),
										PeriodSeconds:       pulumi.Int(15),
									},
									Resources: &corev1.ResourceRequirementsArgs{
										Limits: pulumi.StringMap{
											"nvidia.com/gpu": pulumi.String("1"),
										},
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(32400),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("companion"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(3005),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("discovery"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(5353),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("dlna-tcp"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(32469),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("dlna-udp"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(1900),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("gdm-32410"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32410),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("gdm-32412"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32412),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("gdm-32413"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32413),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("gdm-32414"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32414),
										},
									},
									Env: &corev1.EnvVarArray{
										&corev1.EnvVarArgs{
											Name:  pulumi.String("NVIDIA_VISIBLE_DEVICES"),
											Value: pulumi.String("all"),
										},
										&corev1.EnvVarArgs{
											Name:  pulumi.String("NVIDIA_DRIVER_CAPABILITIES"),
											Value: pulumi.String("compute,video,utility"),
										},
									},
									EnvFrom: &corev1.EnvFromSourceArray{
										&corev1.EnvFromSourceArgs{
											SecretRef: &corev1.SecretEnvSourceArgs{
												Name: secretEnvs.Metadata.Name(),
											},
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("plex-config-volume"),
											MountPath: pulumi.String("/config"),
										},
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("plex-media-nfs-volume"),
											MountPath: pulumi.String("/data"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("plex-config-volume"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: configVolumeClaim.Metadata.Name().Elem(),
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("plex-media-nfs-volume"),
									Nfs: &corev1.NFSVolumeSourceArgs{
										Server:   pulumi.String(config.Require("mediaNfsHost")),
										Path:     pulumi.String(config.Require("mediaNfsPath")),
										ReadOnly: pulumi.Bool(true),
									},
								},
							},
						},
					},
				},
			},
		)

		if err != nil {
			return err
		}

		service, err := corev1.NewService(
			ctx,
			"plex-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace,
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector: pulumi.StringMap{
						"app": pulumi.String("plex"),
					},
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("http"),
							Port:       pulumi.Int(32400),
							TargetPort: pulumi.String("http"),
							Protocol:   pulumi.String("TCP"),
						},
					},
				},
			},
			pulumi.Parent(deployment),
		)

		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(
			ctx,
			"plex-http-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("plex-http"),
					Namespace: namespace,
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("web"),
							pulumi.String("web-secure"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.Sprintf("Host(`%s`)", config.Require("domain")),
								"kind":  pulumi.String("Rule"),
								"services": []kubernetes.UntypedArgs{
									{
										"name": service.Metadata.Name().Elem(),
										"port": pulumi.Int(32400),
									},
								},
							},
						},
						"tls": kubernetes.UntypedArgs{
							"secretName": certificate.OtherFields.ApplyT(func(otherFields interface{}) string {
								fields := otherFields.(map[string]interface{})
								spec := fields["spec"].(map[string]interface{})
								return spec["secretName"].(string)
							}).(pulumi.StringOutput),
						},
					},
				},
			},
			pulumi.Parent(service),
			pulumi.DependsOn([]pulumi.Resource{
				certificate,
			}),
		)

		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(
			ctx,
			"plex-tcp-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRouteTCP"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("plex-tcp"),
					Namespace: namespace,
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("plex"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.String("HostSNI(`*`)"),
								"kind":  pulumi.String("Rule"),
								"services": []kubernetes.UntypedArgs{
									{
										"name": service.Metadata.Name().Elem(),
										"port": pulumi.Int(32400),
									},
								},
							},
						},
					},
				},
			},
			pulumi.Parent(service),
		)

		if err != nil {
			return err
		}

		return nil
	})
}
