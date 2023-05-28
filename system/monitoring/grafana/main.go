package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "grafana")
		namespace := config.Require("namespace")

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = corev1.NewServiceAccount(
			ctx,
			"grafana-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("grafana"),
					Namespace: pulumi.String(namespace),
				},
			},
		); err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app.kubernetes.io/name": pulumi.String("grafana"),
		}

		var deployment *appsv1.Deployment
		if deployment, err = appsv1.NewDeployment(
			ctx,
			"grafana-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    selectorLabels,
					Name:      pulumi.String("grafana"),
					Namespace: pulumi.String(namespace),
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Replicas: pulumi.Int(config.RequireInt("replicas")),
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: selectorLabels,
					},
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Labels: selectorLabels,
						},
						Spec: &corev1.PodSpecArgs{
							ServiceAccountName:           serviceAccount.Metadata.Name(),
							AutomountServiceAccountToken: pulumi.Bool(true),
							SecurityContext: &corev1.PodSecurityContextArgs{
								FsGroup: pulumi.Int(472),
								SupplementalGroups: pulumi.IntArray{
									pulumi.Int(0),
								},
							},
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("grafana"),
									Image: pulumi.String(config.Require("image")),
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											ContainerPort: pulumi.Int(3000),
											Name:          pulumi.String("grafana-http"),
											Protocol:      pulumi.String("TCP"),
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				serviceAccount,
			}),
		); err != nil {
			return err
		}

		// var service *corev1.Service
		if _, err = corev1.NewService(
			ctx,
			"grafana-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    selectorLabels,
					Name:      pulumi.String("grafana"),
					Namespace: pulumi.String(namespace),
				},
				Spec: &corev1.ServiceSpecArgs{
					Selector: selectorLabels,
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("grafana-http"),
							Port:       pulumi.Int(80),
							TargetPort: pulumi.String("grafana-http"),
							Protocol:   pulumi.String("TCP"),
						},
					},
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				deployment,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", pulumi.String(namespace))

		return nil
	})
}
