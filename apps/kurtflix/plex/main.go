package main

import (
	infra "github.com/kurtschwarz/home/packages/infrastructure"
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

		secretEnvs, err := provisionSecretEnvs(ctx, config, namespace)
		if err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("plex"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		_, err = appsv1.NewDeployment(
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
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(32400),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-companion"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(3005),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-discovery"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(5353),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-dlna-tcp"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(32469),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-dlna-udp"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(1900),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-gdm-32410"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32410),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-gdm-32412"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32412),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-gdm-32413"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32413),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex-gdm-32414"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(32414),
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
											Name:      pulumi.String("plex-config-nfs-volume"),
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
									Name: pulumi.String("plex-config-nfs-volume"),
									Nfs: &corev1.NFSVolumeSourceArgs{
										Server:   pulumi.String(config.Require("configNfsHost")),
										Path:     pulumi.String(config.Require("configNfsPath")),
										ReadOnly: pulumi.Bool(false),
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

		return nil
	})
}
