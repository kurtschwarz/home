package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "traefik")

		namespace, err := corev1.NewNamespace(
			ctx,
			"traefik-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("traefik"),
				},
			},
		)

		if err != nil {
			return err
		}

		serviceAccount, err := corev1.NewServiceAccount(
			ctx,
			"traefik-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
			},
			pulumi.Parent(namespace),
		)

		if err != nil {
			return err
		}

		clusterRole, err := rbacv1.NewClusterRole(
			ctx,
			"traefik-cluster-role",
			&rbacv1.ClusterRoleArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
				Rules: rbacv1.PolicyRuleArray{
					rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String(""),
						},
						Resources: pulumi.StringArray{
							pulumi.String("services"),
							pulumi.String("endpoints"),
							pulumi.String("secrets"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},
					rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("extensions"),
							pulumi.String("networking.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("ingresses"),
							pulumi.String("ingressclasses"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},
					rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("extensions"),
							pulumi.String("networking.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("ingresses/status"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("update"),
						},
					},
				},
			},
			pulumi.Parent(namespace),
			pulumi.DependsOn([]pulumi.Resource{
				serviceAccount,
			}),
		)

		if err != nil {
			return err
		}

		clusterRoleBinding, err := rbacv1.NewClusterRoleBinding(
			ctx,
			"traefik-cluster-role-binding",
			&rbacv1.ClusterRoleBindingArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
				RoleRef: &rbacv1.RoleRefArgs{
					ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
					Kind:     pulumi.String("ClusterRole"),
					Name:     pulumi.String("traefik-role"),
				},
				Subjects: &rbacv1.SubjectArray{
					&rbacv1.SubjectArgs{
						Kind:      pulumi.String("ServiceAccount"),
						Name:      serviceAccount.Metadata.Name().Elem(),
						Namespace: namespace.Metadata.Name().Elem(),
					},
				},
			},
			pulumi.Parent(clusterRole),
		)

		if err != nil {
			return err
		}

		configMap, err := corev1.NewConfigMap(
			ctx,
			"traefik-config-map",
			&corev1.ConfigMapArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name().Elem(),
				},
				Data: &pulumi.StringMap{
					"traefik.yaml": pulumi.String(config.Require("traefik.yaml")),
				},
			},
			pulumi.Parent(namespace),
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
					Labels:    selectorLabels,
					Namespace: namespace.Metadata.Name().Elem(),
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
			pulumi.Parent(namespace),
			pulumi.DependsOn([]pulumi.Resource{
				clusterRoleBinding,
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
					Namespace: namespace.Metadata.Name().Elem(),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type: pulumi.String("LoadBalancer"),
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
					},
					Selector: selectorLabels,
				},
			},
			pulumi.Parent(namespace),
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
