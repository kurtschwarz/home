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
		config := config.New(ctx, "sabnzbd")

		namespace, err := corev1.NewNamespace(
			ctx,
			"sabnzbd-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("sabnzbd"),
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
		if _, configVolumeClaim, err = infra.ProvisionLocalVolume(
			ctx,
			namespace.Metadata.Name().Elem(),
			"sabnzbd-config",
			"/mnt/appdata/sabnzbd",
		); err != nil {
			return err
		}

		var mediaVolumeClaim *corev1.PersistentVolumeClaim
		if _, mediaVolumeClaim, err = infra.ProvisionLocalVolume(
			ctx,
			namespace.Metadata.Name().Elem(),
			"sabnzbd-downloads",
			"/mnt/media/Downloads",
		); err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("sabnzbd"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		deployment, err := appsv1.NewDeployment(
			ctx,
			"sabnzbd-deployment",
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
								"kubernetes.io/hostname": pulumi.String("kurtina-k8s-worker-unraid"),
							},
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("sabnzbd"),
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
											Name:      pulumi.String("sabnzbd-config-pv"),
											MountPath: pulumi.String("/config"),
										},
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("sabnzbd-media-pv"),
											MountPath: pulumi.String("/downloads"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("sabnzbd-config-pv"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: configVolumeClaim.Metadata.Name().Elem(),
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("sabnzbd-media-pv"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: mediaVolumeClaim.Metadata.Name().Elem(),
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
			"sabnzbd-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector: pulumi.StringMap{
						"app": pulumi.String("sabnzbd"),
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
			"sabnzbd-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("sabnzbd"),
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

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
