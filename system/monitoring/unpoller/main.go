package main

import (
	infra "github.com/kurtschwarz/home/packages/infrastructure"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiext "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "unpoller")
		namespace := config.Require("namespace")

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = corev1.NewServiceAccount(
			ctx,
			"unpoller-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("unpoller"),
					Namespace: pulumi.String(namespace),
				},
			},
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
			"unpoller-config",
			&corev1.ConfigMapArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("unpoller-config"),
					Namespace: pulumi.String(namespace),
				},
				Data: configMapDataStringMap,
			},
		); err != nil {
			return err
		}

		secretsData := map[string]string{}
		config.RequireObject("secrets", &secretsData)

		secretsDataStringMap := &pulumi.StringMap{}
		for k, v := range secretsData {
			(*secretsDataStringMap)[k] = pulumi.String(v)
		}

		var secrets *corev1.Secret
		if secrets, err = corev1.NewSecret(
			ctx,
			"unpoller-secrets",
			&corev1.SecretArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("unpoller-secrets"),
					Namespace: pulumi.String(namespace),
				},
				StringData: secretsDataStringMap,
			},
		); err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app.kubernetes.io/name": pulumi.String("unpoller"),
		}

		var deployment *appsv1.Deployment
		if deployment, err = appsv1.NewDeployment(
			ctx,
			"unpoller-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    selectorLabels,
					Name:      pulumi.String("unpoller"),
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
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("unpoller"),
									Image: pulumi.String(config.Require("image")),
									EnvFrom: &corev1.EnvFromSourceArray{
										&corev1.EnvFromSourceArgs{
											ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
												Name:     configMap.Metadata.Name(),
												Optional: pulumi.Bool(false),
											},
										},
										&corev1.EnvFromSourceArgs{
											SecretRef: &corev1.SecretEnvSourceArgs{
												Name:     secrets.Metadata.Name(),
												Optional: pulumi.Bool(false),
											},
										},
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("metrics"),
											ContainerPort: pulumi.Int(9130),
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
				configMap,
				secrets,
			}),
		); err != nil {
			return err
		}

		var service *corev1.Service
		if service, err = corev1.NewService(
			ctx,
			"unpoller-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    infra.MergeStringMap(pulumi.StringMap{"app.kubernetes.io/component": pulumi.String("exporter")}, selectorLabels),
					Name:      pulumi.String("unpoller"),
					Namespace: pulumi.String(namespace),
				},
				Spec: &corev1.ServiceSpecArgs{
					ClusterIP: pulumi.String("None"),
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("metrics"),
							Port:       pulumi.Int(9130),
							TargetPort: pulumi.String("metrics"),
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
			"unpoller-service-monitor",
			&apiext.CustomResourceArgs{
				ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
				Kind:       pulumi.String("ServiceMonitor"),
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    infra.MergeStringMap(pulumi.StringMap{"app.kubernetes.io/component": pulumi.String("exporter")}, selectorLabels),
					Name:      pulumi.String("unpoller"),
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
