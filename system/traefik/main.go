package main

import (
	"github.com/kurtschwarz/home/packages/infrastructure"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "traefik")
		namespace := pulumi.String(config.Require("namespace"))

		if err = provisionCrdResources(ctx); err != nil {
			return err
		}

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = provisionRbacResources(ctx, namespace); err != nil {
			return err
		}

		configMap, err := corev1.NewConfigMap(
			ctx,
			"traefik-config-map",
			&corev1.ConfigMapArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: namespace,
				},
				Data: &pulumi.StringMap{
					"traefik.yaml": pulumi.String(config.Require("traefik.yaml")),
				},
			},
		)

		if err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app": pulumi.String("traefik"),
		}

		deployment, err := appsv1.NewDeployment(
			ctx,
			"traefik-deployment",
			&appsv1.DeploymentArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: namespace,
					Name:      pulumi.String("traefik"),
					Labels: infrastructure.MergeStringMap(selectorLabels, pulumi.StringMap{
						"kubernetes.io/cluster-service": pulumi.String("true"),
					}),
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Replicas: pulumi.Int(config.RequireInt("replicas")),
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: selectorLabels,
					},
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Labels:    selectorLabels,
							Namespace: namespace,
						},
						Spec: &corev1.PodSpecArgs{
							Tolerations: &corev1.TolerationArray{
								&corev1.TolerationArgs{
									Key:      pulumi.String("CriticalAddonsOnly"),
									Operator: pulumi.String("Exists"),
								},
								&corev1.TolerationArgs{
									Key:      pulumi.String("node-role.kubernetes.io/control-plane"),
									Operator: pulumi.String("Exists"),
									Effect:   pulumi.String("NoSchedule"),
								},
								&corev1.TolerationArgs{
									Key:      pulumi.String("node-role.kubernetes.io/master"),
									Operator: pulumi.String("Exists"),
									Effect:   pulumi.String("NoSchedule"),
								},
							},
							PriorityClassName: pulumi.String("system-cluster-critical"),
							NodeSelector: pulumi.StringMap{
								"node-role.kubernetes.io/master": pulumi.String("true"),
							},
							ServiceAccountName: serviceAccount.Metadata.Name().Elem(),
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("traefik"),
									Image: pulumi.String(config.Require("image")),
									Args: pulumi.StringArray{
										pulumi.String("--configFile=/etc/traefik/traefik.yaml"),
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("web"),
											ContainerPort: pulumi.Int(80),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("web-secure"),
											ContainerPort: pulumi.Int(443),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("dashboard"),
											ContainerPort: pulumi.Int(8080),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("plex"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(32400),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("syncthing-tcp"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(22000),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("syncthing-udp"),
											Protocol:      pulumi.String("UDP"),
											ContainerPort: pulumi.Int(22000),
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("traefik-config-yaml"),
											MountPath: pulumi.String("/etc/traefik/traefik.yaml"),
											SubPath:   pulumi.String("traefik.yaml"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("traefik-config-yaml"),
									ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
										Name: configMap.Metadata.Name(),
										Items: &corev1.KeyToPathArray{
											&corev1.KeyToPathArgs{
												Key:  pulumi.String("traefik.yaml"),
												Path: pulumi.String("traefik.yaml"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				configMap,
			}),
		)

		if err != nil {
			return err
		}

		_, err = corev1.NewService(
			ctx,
			"traefik-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace,
					Name:      pulumi.String("traefik"),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:           pulumi.String("LoadBalancer"),
					LoadBalancerIP: pulumi.String(config.Require("loadBalancerIP")),
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Port:       pulumi.Int(80),
							Name:       pulumi.String("web"),
							TargetPort: pulumi.String("web"),
						},
						&corev1.ServicePortArgs{
							Port:       pulumi.Int(443),
							Name:       pulumi.String("web-secure"),
							TargetPort: pulumi.String("web-secure"),
						},
						&corev1.ServicePortArgs{
							Port:       pulumi.Int(32400),
							Protocol:   pulumi.String("TCP"),
							Name:       pulumi.String("plex"),
							TargetPort: pulumi.String("plex"),
						},
						&corev1.ServicePortArgs{
							Name:       pulumi.String("syncthing-tcp"),
							Protocol:   pulumi.String("TCP"),
							Port:       pulumi.Int(22000),
							TargetPort: pulumi.String("syncthing-tcp"),
						},
						&corev1.ServicePortArgs{
							Name:       pulumi.String("syncthing-udp"),
							Protocol:   pulumi.String("UDP"),
							Port:       pulumi.Int(22000),
							TargetPort: pulumi.String("syncthing-udp"),
						},
					},
					Selector: selectorLabels,
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				deployment,
			}),
		)

		if err != nil {
			return err
		}

		return nil
	})
}
