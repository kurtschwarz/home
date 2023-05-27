package components

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiext "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PrometheusArgs struct {
	Namespace pulumi.StringInput
}

type Prometheus struct {
	pulumi.ResourceState

	ServiceAccount        *corev1.ServiceAccount
	ServiceAccountRole    *rbacv1.ClusterRole
	ServiceAccountBinding *rbacv1.ClusterRoleBinding

	Instance *apiext.CustomResource
	Service  *corev1.Service
}

func NewPrometheus(
	ctx *pulumi.Context, name string, args *PrometheusArgs, opts ...pulumi.ResourceOption,
) (resource *Prometheus, err error) {
	resource = &Prometheus{}

	if err = ctx.RegisterComponentResource(
		"kurtschwarz:home/system/prometheus:Prometheus",
		name,
		resource,
		opts...,
	); err != nil {
		return nil, err
	}

	if resource.ServiceAccount, err = corev1.NewServiceAccount(
		ctx,
		"prometheus-service-account",
		&corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("prometheus"),
				Namespace: args.Namespace,
			},
		},
	); err != nil {
		return nil, err
	}

	if resource.ServiceAccountRole, err = rbacv1.NewClusterRole(
		ctx,
		"prometheus",
		&rbacv1.ClusterRoleArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("prometheus"),
				Namespace: args.Namespace,
			},
			Rules: &rbacv1.PolicyRuleArray{
				&rbacv1.PolicyRuleArgs{
					ApiGroups: pulumi.StringArray{
						pulumi.String(""),
					},
					Resources: pulumi.StringArray{
						pulumi.String("nodes"),
						pulumi.String("nodes/metrics"),
						pulumi.String("services"),
						pulumi.String("endpoints"),
						pulumi.String("pods"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("get"),
						pulumi.String("list"),
						pulumi.String("watch"),
					},
				},
				&rbacv1.PolicyRuleArgs{
					ApiGroups: pulumi.StringArray{
						pulumi.String(""),
					},
					Resources: pulumi.StringArray{
						pulumi.String("configmaps"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("get"),
					},
				},
				&rbacv1.PolicyRuleArgs{
					ApiGroups: pulumi.StringArray{
						pulumi.String("networking.k8s.io"),
					},
					Resources: pulumi.StringArray{
						pulumi.String("ingresses"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("get"),
						pulumi.String("list"),
						pulumi.String("watch"),
					},
				},
				&rbacv1.PolicyRuleArgs{
					NonResourceURLs: pulumi.StringArray{
						pulumi.String("/metrics"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("get"),
					},
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{
			resource.ServiceAccount,
		}),
	); err != nil {
		return nil, err
	}

	if resource.ServiceAccountBinding, err = rbacv1.NewClusterRoleBinding(
		ctx,
		"prometheus",
		&rbacv1.ClusterRoleBindingArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("prometheus"),
				Namespace: args.Namespace,
			},
			RoleRef: &rbacv1.RoleRefArgs{
				ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
				Kind:     pulumi.String("ClusterRole"),
				Name:     resource.ServiceAccountRole.Metadata.Name().Elem(),
			},
			Subjects: &rbacv1.SubjectArray{
				&rbacv1.SubjectArgs{
					Kind:      pulumi.String("ServiceAccount"),
					Name:      resource.ServiceAccount.Metadata.Name().Elem(),
					Namespace: args.Namespace,
				},
			},
		},
		pulumi.Parent(resource.ServiceAccountRole),
		pulumi.DependsOn([]pulumi.Resource{
			resource.ServiceAccountRole,
		}),
	); err != nil {
		return nil, err
	}

	if resource.Instance, err = apiext.NewCustomResource(
		ctx,
		"prometheus",
		&apiext.CustomResourceArgs{
			ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
			Kind:       pulumi.String("Prometheus"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("prometheus"),
				Namespace: args.Namespace,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"serviceAccountName": resource.ServiceAccount.Metadata.Name(),
					"serviceMonitorSelector": kubernetes.UntypedArgs{
						"matchLabels": pulumi.StringMap{
							"app.kubernetes.io/component": pulumi.String("exporter"),
						},
					},
					"serviceMonitorNamespaceSelector": kubernetes.UntypedArgs{
						"matchNames": pulumi.StringArray{
							args.Namespace,
						},
					},
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{
			resource.ServiceAccount,
		}),
	); err != nil {
		return nil, err
	}

	if resource.Service, err = corev1.NewService(
		ctx,
		"prometheus-service",
		&corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("prometheus"),
				Namespace: args.Namespace,
			},
			Spec: &corev1.ServiceSpecArgs{
				Type: pulumi.String("NodePort"),
				Ports: &corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("web"),
						NodePort:   pulumi.Int(30900),
						Port:       pulumi.Int(80),
						Protocol:   pulumi.String("TCP"),
						TargetPort: pulumi.String("web"),
					},
				},
				Selector: &pulumi.StringMap{
					"prometheus": pulumi.String("prometheus"),
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{
			resource.Instance,
		}),
	); err != nil {
		return nil, err
	}

	return resource, nil
}
