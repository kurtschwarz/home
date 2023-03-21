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
		config := config.New(ctx, "tautulli")
		namespace := infra.RequireNamespace(ctx, "homelab-apps-kurtflix", ctx.Stack())

		certificate, err := infra.ProvisionCertificate(ctx, namespace, config.Require("domain"))
		if err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("tautulli"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		deployment, err := appsv1.NewDeployment(
			ctx,
			"tautulli-deployment",
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
									Name:  pulumi.String("tautulli"),
									Image: pulumi.String(config.Require("image")),
									ReadinessProbe: &corev1.ProbeArgs{
										TcpSocket: &corev1.TCPSocketActionArgs{
											Port: pulumi.Int(config.RequireInt("port")),
										},
										InitialDelaySeconds: pulumi.Int(20),
										PeriodSeconds:       pulumi.Int(15),
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(config.RequireInt("port")),
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("tautulli-config-nfs-volume"),
											MountPath: pulumi.String("/config"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("tautulli-config-nfs-volume"),
									Nfs: &corev1.NFSVolumeSourceArgs{
										Server:   pulumi.String(config.Require("configNfsHost")),
										Path:     pulumi.String(config.Require("configNfsPath")),
										ReadOnly: pulumi.Bool(false),
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
			"tautulli-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace,
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector: pulumi.StringMap{
						"app": pulumi.String("tautulli"),
					},
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("http"),
							Port:       pulumi.Int(config.RequireInt("port")),
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
			"tautulli-middleware-x-headers",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("Middleware"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("tautulli-middleware-x-headers"),
					Namespace: namespace,
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"headers": kubernetes.UntypedArgs{
							"customRequestHeaders": kubernetes.UntypedArgs{
								"X-Forwarded-Host":  pulumi.String(config.Require("domain")),
								"X-Forwarded-Proto": pulumi.String("https"),
								"X-Forwarded-Ssl":   pulumi.String("on"),
							},
						},
					},
				},
			},
		)

		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(
			ctx,
			"tautulli-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("tautulli"),
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
										"port": pulumi.Int(config.RequireInt("port")),
									},
								},
								"middlewares": []kubernetes.UntypedArgs{
									{
										"name":      "tautulli-middleware-x-headers",
										"namespace": namespace,
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

		return nil
	})
}
