package main

import (
	"regexp"

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
		config := config.New(ctx, "ntfy")
		version := regexp.MustCompile(`(?m):v?(?P<version>.+)@`).FindStringSubmatch(config.Require("image"))[1]

		var namespace *corev1.Namespace
		if namespace, err = corev1.NewNamespace(
			ctx,
			"ntfy-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("ntfy"),
				},
			},
		); err != nil {
			return err
		}

		var serviceAccount *corev1.ServiceAccount
		if serviceAccount, err = corev1.NewServiceAccount(
			ctx,
			"ntfy-service-account",
			&corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("ntfy"),
					Namespace: namespace.Metadata.Name(),
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var configMap *corev1.ConfigMap
		if configMap, err = corev1.NewConfigMap(
			ctx,
			"ntfy-config",
			&corev1.ConfigMapArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("ntfy-config"),
					Namespace: namespace.Metadata.Name(),
				},
				Data: pulumi.StringMap{
					"server.yml": pulumi.String(config.Require("server.yml")),
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var cacheVolume *infra.LonghornVolume
		if cacheVolume, err = infra.NewLonghornVolume(
			ctx,
			"ntfy-cache",
			&infra.LonghornVolumeArgs{
				Size:       pulumi.String("10Gi"),
				AccessMode: infra.ReadWriteOnce,
				Namespace:  namespace.Metadata.Name().Elem(),
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var certificate *apiext.CustomResource
		if certificate, err = infra.ProvisionCertificate(
			ctx,
			namespace.Metadata.Name().Elem(),
			config.Require("domain"),
		); err != nil {
			return err
		}

		selectorLabels := pulumi.StringMap{
			"app.kubernetes.io/name": pulumi.String("ntfy"),
		}

		deploymentLabels := infra.MergeStringMap(selectorLabels, pulumi.StringMap{
			"app.kubernetes.io/version": pulumi.String(version),
		})

		var deployment *appsv1.Deployment
		if deployment, err = appsv1.NewDeployment(
			ctx,
			"ntfy-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("ntfy"),
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
									Name:            pulumi.String("ntfy"),
									Image:           pulumi.String(config.Require("image")),
									ImagePullPolicy: pulumi.String("Always"),
									Args: &pulumi.StringArray{
										pulumi.String("serve"),
									},
									Resources: &corev1.ResourceRequirementsArgs{
										Limits: &pulumi.StringMap{
											"memory": pulumi.String("128Mi"),
											"cpu":    pulumi.String("500m"),
										},
									},
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(80),
										},
									},
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("ntfy-config-yaml"),
											MountPath: pulumi.String("/etc/ntfy/server.yml"),
											SubPath:   pulumi.String("server.yml"),
										},
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("ntfy-cache-volume"),
											MountPath: pulumi.String("/var/cache/ntfy"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("ntfy-config-yaml"),
									ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
										Name: configMap.Metadata.Name(),
										Items: &corev1.KeyToPathArray{
											&corev1.KeyToPathArgs{
												Key:  pulumi.String("server.yml"),
												Path: pulumi.String("server.yml"),
											},
										},
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("ntfy-cache-volume"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: cacheVolume.PersistentVolumeClaim.Metadata.Name().Elem(),
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
				configMap,
				cacheVolume,
				serviceAccount,
			}),
		); err != nil {
			return err
		}

		var service *corev1.Service
		if service, err = corev1.NewService(
			ctx,
			"ntfy-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("ntfy"),
					Namespace: namespace.Metadata.Name(),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector:        selectorLabels,
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("http"),
							Port:       pulumi.Int(80),
							TargetPort: pulumi.String("http"),
							Protocol:   pulumi.String("TCP"),
						},
					},
				},
			},
			pulumi.Parent(namespace),
			pulumi.DependsOn([]pulumi.Resource{
				certificate,
				deployment,
			}),
		); err != nil {
			return err
		}

		if _, err = apiext.NewCustomResource(
			ctx,
			"ntfy-http-ingress-route",
			&apiext.CustomResourceArgs{
				ApiVersion: pulumi.String("traefik.containo.us/v1alpha1"),
				Kind:       pulumi.String("IngressRoute"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("ntfy-http"),
					Namespace: namespace.Metadata.Name(),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"entryPoints": pulumi.StringArray{
							pulumi.String("web"),
							pulumi.String("web-secure"),
						},
						"routes": []kubernetes.UntypedArgs{
							{
								"match": pulumi.Sprintf("Host(`%s`)", config.Require("domain")),
								"kind":  pulumi.String("Rule"),
								"services": []kubernetes.UntypedArgs{
									{
										"name": service.Metadata.Name().Elem(),
										"port": pulumi.Int(80),
									},
								},
							},
						},
						"tls": kubernetes.UntypedArgs{
							"secretName": certificate.OtherFields.ApplyT(func(otherFields interface{}) string {
								fields := otherFields.(map[string]interface{})
								spec := fields["spec"].(map[string]interface{})
								return spec["secretName"].(string)
							}).(pulumi.StringOutput),
						},
					},
				},
			},
			pulumi.Parent(service),
			pulumi.DependsOn([]pulumi.Resource{
				certificate,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())
		ctx.Export("version", pulumi.String(version))

		return nil
	})
}
