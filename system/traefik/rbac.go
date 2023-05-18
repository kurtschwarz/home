package main

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func provisionRbacResources(ctx *pulumi.Context, namespace pulumi.String) (serviceAccount *corev1.ServiceAccount, err error) {
	if serviceAccount, err = corev1.NewServiceAccount(
		ctx,
		"traefik-service-account",
		&corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: namespace,
			},
		},
	); err != nil {
		return nil, err
	}

	var clusterRole *rbacv1.ClusterRole
	if clusterRole, err = rbacv1.NewClusterRole(
		ctx,
		"traefik-cluster-role",
		&rbacv1.ClusterRoleArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: namespace,
				Name:      pulumi.String("traefik-role"),
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
				rbacv1.PolicyRuleArgs{
					ApiGroups: pulumi.StringArray{
						pulumi.String("traefik.containo.us"),
					},
					Resources: pulumi.StringArray{
						pulumi.String("ingressroutes"),
						pulumi.String("ingressroutetcps"),
						pulumi.String("ingressrouteudps"),
						pulumi.String("middlewares"),
						pulumi.String("middlewaretcps"),
						pulumi.String("tlsoptions"),
						pulumi.String("tlsstores"),
						pulumi.String("traefikservices"),
						pulumi.String("serverstransports"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("get"),
						pulumi.String("list"),
						pulumi.String("watch"),
					},
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{
			serviceAccount,
		}),
	); err != nil {
		return nil, err
	}

	if _, err = rbacv1.NewClusterRoleBinding(
		ctx,
		"traefik-cluster-role-binding",
		&rbacv1.ClusterRoleBindingArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: namespace,
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
					Namespace: namespace,
				},
			},
		},
		pulumi.Parent(clusterRole),
	); err != nil {
		return nil, err
	}

	return serviceAccount, nil
}
