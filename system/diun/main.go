package main

import (
	"regexp"

	infra "github.com/kurtschwarz/home/packages/infrastructure"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "diun")
		version := regexp.MustCompile(`(?m):(?P<version>.+)@`).FindStringSubmatch(config.Require("image"))[1]

		var namespace *corev1.Namespace
		if namespace, err = corev1.NewNamespace(
			ctx,
			"diun-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("diun"),
				},
			},
		); err != nil {
			return err
		}

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = corev1.NewServiceAccount(
			ctx,
			"diun-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("diun"),
					Namespace: namespace.Metadata.Name(),
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var clusterRole *rbacv1.ClusterRole
		if clusterRole, err = rbacv1.NewClusterRole(
			ctx,
			"diun-cluster-role",
			&rbacv1.ClusterRoleArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("diun"),
				},
				Rules: &rbacv1.PolicyRuleArray{
					&rbacv1.PolicyRuleArgs{
						ApiGroups: &pulumi.StringArray{
							pulumi.String(""),
						},
						Resources: &pulumi.StringArray{
							pulumi.String("pods"),
						},
						Verbs: &pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("watch"),
						},
					},
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		if _, err = rbacv1.NewClusterRoleBinding(
			ctx,
			"diun-cluster-role-binding",
			&rbacv1.ClusterRoleBindingArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("diun"),
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

		var dataVolume *infra.LonghornVolume
		if dataVolume, err = infra.NewLonghornVolume(
			ctx,
			"diun-data",
			&infra.LonghornVolumeArgs{
				Size:       pulumi.String("5Gi"),
				AccessMode: infra.ReadWriteOnce,
				Namespace:  namespace.Metadata.Name().Elem(),
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		configMapData := map[string]string{}
		config.RequireObject("config", &configMapData)

		configMapDataStringMap := &pulumi.StringMap{}
		for k, v := range configMapData {
			(*configMapDataStringMap)[k] = pulumi.String(v)
		}

		var configMap *corev1.ConfigMap
		if configMap, err = corev1.NewConfigMap(
			ctx,
			"diun-config",
			&corev1.ConfigMapArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("diun-config"),
					Namespace: namespace.Metadata.Name(),
				},
				Data: configMapDataStringMap,
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app.kubernetes.io/name": pulumi.String("diun"),
		}

		deploymentLabels := infra.MergeStringMap(selectorLabels, pulumi.StringMap{
			"app.kubernetes.io/version": pulumi.String(version),
		})

		if _, err = appsv1.NewDeployment(
			ctx,
			"diun",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("diun"),
					Namespace: namespace.Metadata.Name(),
					Labels:    deploymentLabels,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: selectorLabels,
					},
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Labels: deploymentLabels,
							Annotations: &pulumi.StringMap{
								"diun.enable": pulumi.String("true"),
							},
						},
						Spec: &corev1.PodSpecArgs{
							ServiceAccountName:           serviceAccount.Metadata.Name(),
							AutomountServiceAccountToken: pulumi.Bool(true),
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:            pulumi.String("diun"),
									Image:           pulumi.String(config.Require("image")),
									ImagePullPolicy: pulumi.String("Always"),
									Args: pulumi.StringArray{
										pulumi.String("serve"),
									},
									EnvFrom: &corev1.EnvFromSourceArray{
										&corev1.EnvFromSourceArgs{
											ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
												Name:     configMap.Metadata.Name(),
												Optional: pulumi.Bool(false),
											},
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("diun-data-volume"),
											MountPath: pulumi.String("/data"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("diun-data-volume"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: dataVolume.PersistentVolumeClaim.Metadata.Name().Elem(),
									},
								},
							},
							RestartPolicy: pulumi.String("Always"),
						},
					},
				},
			},
			pulumi.Parent(namespace),
			pulumi.DependsOn([]pulumi.Resource{
				dataVolume,
				configMap,
				serviceAccount,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())
		ctx.Export("version", pulumi.String(version))

		return nil
	})
}
