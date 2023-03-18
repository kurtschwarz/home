package main

import (
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
		config := config.New(ctx, "grafana")

		deploymentLabels := pulumi.StringMap{"app": pulumi.String("grafana")}
		deployment, err := appsv1.NewDeployment(
			ctx,
			"grafana-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: deploymentLabels,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: deploymentLabels,
					},
					Template: corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Labels: deploymentLabels,
						},
						Spec: corev1.PodSpecArgs{
							SecurityContext: &corev1.PodSecurityContextArgs{
								FsGroup: pulumi.Int(472),
								SupplementalGroups: pulumi.IntArray{
									pulumi.Int(0),
								},
							},
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:            pulumi.String("grafana"),
									Image:           pulumi.String(config.Require("image")),
									ImagePullPolicy: pulumi.String("IfNotPresent"),
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											ContainerPort: pulumi.Int(config.RequireInt("port")),
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
		)

		if err != nil {
			return err
		}

		service, err := corev1.NewService(
			ctx,
			"grafana-service",
			&corev1.ServiceArgs{
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector: pulumi.StringMap{
						"app": pulumi.String("grafana"),
					},
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("grafana-http"),
							Port:       pulumi.Int(config.RequireInt("port")),
							TargetPort: pulumi.String("grafana-http"),
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
			"grafana-ingress-route",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("grafana"),
					Namespace: pulumi.String("default"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("web"),
							pulumi.String("web-secure"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.String("Host(`grafana.kurtina.ca`)"),
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
							"secretName": pulumi.String("grafana-kurtina-ca"),
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
