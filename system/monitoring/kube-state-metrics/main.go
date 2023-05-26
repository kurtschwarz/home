package main

import (
	infra "github.com/kurtschwarz/home/packages/infrastructure"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiext "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "kubeStateMetrics")
		namespace := config.Require("namespace")

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = corev1.NewServiceAccount(
			ctx,
			"kube-state-metrics-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("kube-state-metrics"),
					Namespace: pulumi.String(namespace),
				},
			},
		); err != nil {
			return err
		}

		var clusterRole *rbacv1.ClusterRole
		if clusterRole, err = rbacv1.NewClusterRole(
			ctx,
			"kube-state-metrics-cluster-role",
			&rbacv1.ClusterRoleArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("kube-state-metrics"),
					Namespace: pulumi.String(namespace),
				},
				Rules: &rbacv1.PolicyRuleArray{
					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String(""),
						},
						Resources: pulumi.StringArray{
							pulumi.String("configmaps"),
							pulumi.String("secrets"),
							pulumi.String("nodes"),
							pulumi.String("pods"),
							pulumi.String("services"),
							pulumi.String("serviceaccounts"),
							pulumi.String("resourcequotas"),
							pulumi.String("replicationcontrollers"),
							pulumi.String("limitranges"),
							pulumi.String("persistentvolumeclaims"),
							pulumi.String("persistentvolumes"),
							pulumi.String("namespaces"),
							pulumi.String("endpoints"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("apps"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("statefulsets"),
							pulumi.String("daemonsets"),
							pulumi.String("deployments"),
							pulumi.String("replicasets"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("batch"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("cronjobs"),
							pulumi.String("jobs"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("autoscaling"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("horizontalpodautoscalers"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("authentication.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("tokenreviews"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("create"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("authorization.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("subjectaccessreviews"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("create"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("policy"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("poddisruptionbudgets"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("certificates.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("certificatesigningrequests"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("discovery.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("endpointslices"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("storage.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("storageclasses"),
							pulumi.String("volumeattachments"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("admissionregistration.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("mutatingwebhookconfigurations"),
							pulumi.String("validatingwebhookconfigurations"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("networking.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("networkpolicies"),
							pulumi.String("ingressclasses"),
							pulumi.String("ingresses"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("coordination.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("leases"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},

					&rbacv1.PolicyRuleArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("rbac.authorization.k8s.io"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("clusterrolebindings"),
							pulumi.String("clusterroles"),
							pulumi.String("rolebindings"),
							pulumi.String("roles"),
						},
						Verbs: pulumi.StringArray{
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},
				},
			},
		); err != nil {
			return err
		}

		if _, err = rbacv1.NewClusterRoleBinding(
			ctx,
			"kube-state-metrics-cluster-role-binding",
			&rbacv1.ClusterRoleBindingArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("kube-state-metrics"),
					Namespace: pulumi.String(namespace),
				},
				RoleRef: &rbacv1.RoleRefArgs{
					ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
					Kind:     pulumi.String("ClusterRole"),
					Name:     clusterRole.Metadata.Name().Elem(),
				},
				Subjects: &rbacv1.SubjectArray{
					&rbacv1.SubjectArgs{
						Kind:      pulumi.String("ServiceAccount"),
						Name:      serviceAccount.Metadata.Name().Elem(),
						Namespace: serviceAccount.Metadata.Namespace().Elem(),
					},
				},
			},
			pulumi.Parent(clusterRole),
			pulumi.DependsOn([]pulumi.Resource{
				serviceAccount,
			}),
		); err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app.kubernetes.io/name": pulumi.String("kube-state-metrics"),
		}

		var deployment *appsv1.Deployment
		if deployment, err = appsv1.NewDeployment(
			ctx,
			"kube-state-metrics-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    selectorLabels,
					Name:      pulumi.String("kube-state-metrics"),
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
							ServiceAccountName:           serviceAccount.Metadata.Name().Elem(),
							AutomountServiceAccountToken: pulumi.Bool(true),
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("kube-state-metrics"),
									Image: pulumi.String(config.Require("image")),
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("metrics"),
											ContainerPort: pulumi.Int(8080),
										},
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("telemetry"),
											ContainerPort: pulumi.Int(8081),
										},
									},
									ReadinessProbe: &corev1.ProbeArgs{
										HttpGet: &corev1.HTTPGetActionArgs{
											Path: pulumi.String("/"),
											Port: pulumi.Int(8081),
										},
										InitialDelaySeconds: pulumi.Int(5),
										TimeoutSeconds:      pulumi.Int(5),
									},
									LivenessProbe: &corev1.ProbeArgs{
										HttpGet: &corev1.HTTPGetActionArgs{
											Path: pulumi.String("healthz"),
											Port: pulumi.Int(8080),
										},
										InitialDelaySeconds: pulumi.Int(5),
										TimeoutSeconds:      pulumi.Int(5),
									},
									SecurityContext: &corev1.SecurityContextArgs{
										AllowPrivilegeEscalation: pulumi.Bool(false),
										Capabilities: &corev1.CapabilitiesArgs{
											Drop: &pulumi.StringArray{
												pulumi.String("ALL"),
											},
										},
										ReadOnlyRootFilesystem: pulumi.Bool(true),
										RunAsNonRoot:           pulumi.Bool(true),
										RunAsUser:              pulumi.Int(65534),
										SeccompProfile: &corev1.SeccompProfileArgs{
											Type: pulumi.String("RuntimeDefault"),
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

		var service *corev1.Service
		if service, err = corev1.NewService(
			ctx,
			"kube-state-metrics-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    infra.MergeStringMap(pulumi.StringMap{"app.kubernetes.io/component": pulumi.String("exporter")}, selectorLabels),
					Name:      pulumi.String("kube-state-metrics"),
					Namespace: pulumi.String(namespace),
				},
				Spec: &corev1.ServiceSpecArgs{
					ClusterIP: pulumi.String("None"),
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("metrics"),
							Port:       pulumi.Int(8080),
							TargetPort: pulumi.String("metrics"),
						},
						&corev1.ServicePortArgs{
							Name:       pulumi.String("telemetry"),
							Port:       pulumi.Int(8081),
							TargetPort: pulumi.String("telemetry"),
						},
					},
					Selector: selectorLabels,
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				deployment,
			}),
		); err != nil {
			return err
		}

		if _, err = apiext.NewCustomResource(
			ctx,
			"kube-state-metrics-service-monitor",
			&apiext.CustomResourceArgs{
				ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
				Kind:       pulumi.String("ServiceMonitor"),
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    infra.MergeStringMap(pulumi.StringMap{"app.kubernetes.io/component": pulumi.String("exporter")}, selectorLabels),
					Name:      pulumi.String("kube-state-metrics"),
					Namespace: pulumi.String(namespace),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"endpoints": []kubernetes.UntypedArgs{
							{
								"interval": pulumi.String("30s"),
								"port":     pulumi.String("metrics"),
							},
						},
						"selector": kubernetes.UntypedArgs{
							"matchLabels": selectorLabels,
						},
					},
				},
			},
			pulumi.Parent(service),
		); err != nil {
			return err
		}

		ctx.Export("namespace", pulumi.String(namespace))

		return nil
	})
}
