package main

import (
	"fmt"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func provisionVolumes(ctx *pulumi.Context, namespace pulumi.StringInput) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	persistentVolume, err := corev1.NewPersistentVolume(
		ctx,
		"grafana-pv",
		&corev1.PersistentVolumeArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("grafana-pv"),
				Labels: pulumi.StringMap{
					"app":  pulumi.String("grafana"),
					"type": pulumi.String("local"),
				},
				Namespace: namespace,
			},
			Spec: &corev1.PersistentVolumeSpecArgs{
				StorageClassName: pulumi.String("microk8s-hostpath"),
				Capacity: pulumi.StringMap{
					"storage": pulumi.String("20Gi"),
				},
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				HostPath: &corev1.HostPathVolumeSourceArgs{
					Path: pulumi.String("/mnt/appdata/grafana/data"),
				},
			},
		},
	)

	if err != nil {
		return nil, nil, err
	}

	persistentVolumeClaim, err := corev1.NewPersistentVolumeClaim(
		ctx,
		"grafana-pv-claim",
		&corev1.PersistentVolumeClaimArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("grafana-pv-claim"),
				Labels: pulumi.StringMap{
					"app": pulumi.String("grafana"),
				},
				Namespace: namespace,
			},
			Spec: &corev1.PersistentVolumeClaimSpecArgs{
				StorageClassName: pulumi.String("microk8s-hostpath"),
				Resources: &corev1.ResourceRequirementsArgs{
					Requests: pulumi.StringMap{
						"storage": pulumi.String("20Gi"),
					},
				},
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{
			persistentVolume,
		}),
	)

	if err != nil {
		return nil, nil, err
	}

	return persistentVolume, persistentVolumeClaim, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "grafana")

		namespaceRef, err := pulumi.NewStackReference(
			ctx,
			fmt.Sprintf("kurtschwarz/homelab-system-monitoring/%s", ctx.Stack()),
			nil,
		)

		if err != nil {
			return err
		}

		namespace := namespaceRef.GetOutput(pulumi.String("namespace")).AsStringOutput()

		persistentVolume, persistentVolumeClaim, err := provisionVolumes(
			ctx,
			namespace,
		)

		if err != nil {
			return err
		}

		certificate, err := apiextensions.NewCustomResource(ctx, "grafana-kurtina-ca-cert", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("cert-manager.io/v1"),
			Kind:       pulumi.String("Certificate"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("grafana-kurtina-ca-cert"),
				Namespace: namespace,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"commonName": pulumi.String("grafana.kurtina.ca"),
					"secretName": pulumi.String("grafana-kurtina-ca-cert"),
					"secretTemplate": kubernetes.UntypedArgs{
						"namespace": namespace,
					},
					"dnsNames": pulumi.StringArray{
						pulumi.String("grafana.kurtina.ca"),
					},
					"issuerRef": kubernetes.UntypedArgs{
						"name": pulumi.String("cert-manager-lets-encrypt-issuer"),
						"kind": pulumi.String("ClusterIssuer"),
					},
				},
			},
		})

		if err != nil {
			return err
		}

		deploymentLabels := pulumi.StringMap{"app": pulumi.String("grafana")}
		deployment, err := appsv1.NewDeployment(
			ctx,
			"grafana-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:    deploymentLabels,
					Namespace: namespace,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: deploymentLabels,
					},
					Template: corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Labels:    deploymentLabels,
							Namespace: namespace,
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
									VolumeMounts: &corev1.VolumeMountArray{
										&corev1.VolumeMountArgs{
											Name:      pulumi.String("grafana-pv"),
											MountPath: pulumi.String("/var/lib/grafana"),
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("grafana-pv"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: persistentVolumeClaim.Metadata.Name().Elem(),
									},
								},
							},
						},
					},
				},
			},
			pulumi.DependsOn([]pulumi.Resource{
				persistentVolume,
			}),
		)

		if err != nil {
			return err
		}

		service, err := corev1.NewService(
			ctx,
			"grafana-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace,
				},
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
					Namespace: namespace,
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
							"secretName": pulumi.String("grafana-kurtina-ca-cert"),
						},
					},
				},
			},
			pulumi.Parent(service),
			pulumi.DependsOn([]pulumi.Resource{
				certificate,
			}),
		)

		if err != nil {
			return err
		}

		return nil
	})
}