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

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "syncthing")

		namespace, err := corev1.NewNamespace(
			ctx,
			"syncthing-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("syncthing"),
				},
			},
		)

		if err != nil {
			return err
		}

		certificate, err := infra.ProvisionCertificate(
			ctx,
			namespace.Metadata.Name().Elem(),
			config.Require("domain"),
		)

		if err != nil {
			return err
		}

		var configVolumeClaim *corev1.PersistentVolumeClaim
		if configVolumeClaim, err = corev1.NewPersistentVolumeClaim(
			ctx,
			"syncthing-config-pvc",
			&corev1.PersistentVolumeClaimArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("syncthing-config-pvc"),
					Namespace: namespace.Metadata.Name().Elem(),
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

		var syncVolumeClaim *corev1.PersistentVolumeClaim
		if _, syncVolumeClaim, err = infra.ProvisionLocalVolume(
			ctx,
			namespace.Metadata.Name().Elem(),
			"syncthing-sync",
			"/mnt/media/Sync",
		); err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("syncthing"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		deployment, err := appsv1.NewDeployment(
			ctx,
			"syncthing-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
					Labels:    deploymentLabels,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: deploymentLabels,
					},
					Replicas: pulumi.Int(config.RequireInt("replicas")),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Namespace: namespace.Metadata.Name().Elem(),
							Labels:    deploymentLabels,
						},
						Spec: &corev1.PodSpecArgs{
							NodeSelector: pulumi.StringMap{
								"kubernetes.io/hostname": pulumi.String("worker-02.k3s.kurtina.ca"),
							},
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("syncthing"),
									Image: pulumi.String(config.Require("image")),
									ReadinessProbe: &corev1.ProbeArgs{
										TcpSocket: &corev1.TCPSocketActionArgs{
											Port: pulumi.Int(config.RequireInt("port")),
										},
										InitialDelaySeconds: pulumi.Int(20),
										PeriodSeconds:       pulumi.Int(15),
									},
									Env: &corev1.EnvVarArray{
										&corev1.EnvVarArgs{
											Name:  pulumi.String("HOSTNAME"),
											Value: pulumi.String(""),
										},
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(config.RequireInt("port")),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("sync-tcp"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(22000),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("sync-udp"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(22000),
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("syncthing-config-pv"),
											MountPath: pulumi.String("/config"),
										},
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("syncthing-sync-pv"),
											MountPath: pulumi.String("/mnt/sync"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("syncthing-config-pv"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: configVolumeClaim.Metadata.Name().Elem(),
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("syncthing-sync-pv"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: syncVolumeClaim.Metadata.Name().Elem(),
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
			"syncthing-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector: pulumi.StringMap{
						"app": pulumi.String("syncthing"),
					},
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("http"),
							Port:       pulumi.Int(config.RequireInt("port")),
							TargetPort: pulumi.String("http"),
							Protocol:   pulumi.String("TCP"),
						},
						&corev1.ServicePortArgs{
							Name:       pulumi.String("sync-tcp"),
							Port:       pulumi.Int(22000),
							TargetPort: pulumi.String("sync-tcp"),
							Protocol:   pulumi.String("TCP"),
						},
						&corev1.ServicePortArgs{
							Name:       pulumi.String("sync-udp"),
							Port:       pulumi.Int(22000),
							TargetPort: pulumi.String("sync-udp"),
							Protocol:   pulumi.String("UDP"),
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
			"syncthing-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("syncthing"),
					Namespace: namespace.Metadata.Name().Elem(),
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
										"port": pulumi.Int(config.RequireInt("port")),
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
			"syncthing-tcp-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRouteTCP"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("syncthing-sync-tcp"),
					Namespace: namespace.Metadata.Name().Elem(),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("syncthing-tcp"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.String("HostSNI(`*`)"),
								"kind":  pulumi.String("Rule"),
								"services": []kubernetes.UntypedArgs{
									{
										"name": service.Metadata.Name().Elem(),
										"port": pulumi.Int(22000),
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

		_, err = apiextensions.NewCustomResource(
			ctx,
			"syncthing-udp-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRouteUDP"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("syncthing-sync-udp"),
					Namespace: namespace.Metadata.Name().Elem(),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("syncthing-udp"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.String("HostSNI(`*`)"),
								"kind":  pulumi.String("Rule"),
								"services": []kubernetes.UntypedArgs{
									{
										"name": service.Metadata.Name().Elem(),
										"port": pulumi.Int(22000),
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

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
