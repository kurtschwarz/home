package main

import (
	"os"
	"path/filepath"
	"strings"

	doublestar "github.com/bmatcuk/doublestar/v4"
	infra "github.com/kurtschwarz/home/packages/infrastructure"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func generateConfigMapData(config *config.Config) (data *pulumi.StringMap, err error) {
	var cwd string
	if cwd, err = os.Getwd(); err != nil {
		return nil, err
	}

	data = &pulumi.StringMap{}
	configRoot := filepath.Join(cwd, "config")

	var files []string
	if files, err = doublestar.Glob(os.DirFS(configRoot), "**/*.yaml"); err != nil {
		return nil, err
	}

	for _, filePath := range files {
		var fileContents []byte
		if fileContents, err = os.ReadFile(filepath.Join(configRoot, filePath)); err != nil {
			return nil, err
		}

		(*data)[strings.ReplaceAll(filePath, "/", "__")] = pulumi.String(fileContents)
	}

	return data, nil
}

func generateConfigMapMounts(configMapData *pulumi.StringMap, mounts *corev1.VolumeMountArray) *corev1.VolumeMountArray {
	configMapMounts := corev1.VolumeMountArray{}

	for fileKey := range *configMapData {
		configMapMounts = append(configMapMounts, &corev1.VolumeMountArgs{
			Name:      pulumi.String("home-assistant-config-map-mount"),
			MountPath: pulumi.String(strings.ReplaceAll(filepath.Join("config", fileKey), "__", "/")),
			SubPath:   pulumi.String(strings.ReplaceAll(fileKey, "__", "/")),
			ReadOnly:  pulumi.Bool(true),
		})
	}

	return infra.MergeVolumeMountArray(mounts, &configMapMounts)
}

func generateConfigMapItems(configMapData *pulumi.StringMap) *corev1.KeyToPathArray {
	configMapItems := corev1.KeyToPathArray{}

	for fileKey := range *configMapData {
		configMapItems = append(configMapItems, &corev1.KeyToPathArgs{
			Key:  pulumi.String(fileKey),
			Path: pulumi.String(strings.ReplaceAll(fileKey, "__", "/")),
		})
	}

	return &configMapItems
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "home-assistant")

		var namespace *corev1.Namespace
		if namespace, err = corev1.NewNamespace(
			ctx,
			"home-assistant-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("home-assistant"),
				},
			},
		); err != nil {
			return err
		}

		var configVolumeClaim *corev1.PersistentVolumeClaim
		if configVolumeClaim, err = corev1.NewPersistentVolumeClaim(
			ctx,
			"home-assistant-config-pvc",
			&corev1.PersistentVolumeClaimArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("home-assistant-config-pvc"),
					Namespace: namespace.Metadata.Name(),
					Annotations: &pulumi.StringMap{
						"pulumi.com/skipAwait": pulumi.String("true"),
					},
				},
				Spec: &corev1.PersistentVolumeClaimSpecArgs{
					StorageClassName: pulumi.String("longhorn"),
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
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var configMapData *pulumi.StringMap
		if configMapData, err = generateConfigMapData(config); err != nil {
			return err
		}

		var configMap *corev1.ConfigMap
		if configMap, err = corev1.NewConfigMap(
			ctx,
			"home-assistant-config-map",
			&corev1.ConfigMapArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name(),
					Name:      pulumi.String("home-assistant-config"),
				},
				Data: configMapData,
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		secretsMap := map[string]string{}
		config.RequireObject("secrets", &secretsMap)

		secretsStringMap := &pulumi.StringMap{}
		for k, v := range secretsMap {
			(*secretsStringMap)[k] = pulumi.String(v)
		}

		var secrets *corev1.Secret
		if secrets, err = corev1.NewSecret(
			ctx,
			"home-assistant-secrets",
			&corev1.SecretArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name(),
					Name:      pulumi.String("home-assistant-secrets"),
				},
				StringData: secretsStringMap,
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		sharedLabels := pulumi.StringMap{
			"app": pulumi.String("home-assistant"),
		}

		deploymentLabels := infra.MergeStringMap(
			sharedLabels,
			pulumi.StringMap{},
		)

		var deployment *appsv1.Deployment
		if deployment, err = appsv1.NewDeployment(
			ctx,
			"home-assistant-deployment",
			&appsv1.DeploymentArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name(),
					Name:      pulumi.String("home-assistant-deployment"),
					Labels:    deploymentLabels,
				},
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{
						MatchLabels: deploymentLabels,
					},
					Replicas: pulumi.Int(config.RequireInt("replicas")),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Namespace: namespace.Metadata.Name(),
							Labels:    deploymentLabels,
						},
						Spec: &corev1.PodSpecArgs{
							Containers: &corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String("home-assistant"),
									Image: pulumi.String(config.Require("image")),
									Ports: &corev1.ContainerPortArray{
										&corev1.ContainerPortArgs{
											Name:          pulumi.String("http"),
											Protocol:      pulumi.String("TCP"),
											ContainerPort: pulumi.Int(8123),
										},
									},
									VolumeMounts: generateConfigMapMounts(
										configMapData,
										&corev1.VolumeMountArray{
											&corev1.VolumeMountArgs{
												Name:      pulumi.String("home-assistant-config-pv-mount"),
												MountPath: pulumi.String("/config"),
												ReadOnly:  pulumi.Bool(false),
											},
											&corev1.VolumeMountArgs{
												Name:      pulumi.String("home-assistant-secrets-mount"),
												MountPath: pulumi.String("/ssl/lutron"),
												ReadOnly:  pulumi.Bool(true),
											},
										},
									),
									SecurityContext: &corev1.SecurityContextArgs{
										Capabilities: &corev1.CapabilitiesArgs{
											Add: pulumi.StringArray{
												pulumi.String("NET_ADMIN"),
												pulumi.String("NET_RAW"),
												pulumi.String("NET_BROADCAST"),
											},
										},
									},
								},
							},
							Volumes: &corev1.VolumeArray{
								&corev1.VolumeArgs{
									Name: pulumi.String("home-assistant-config-pv-mount"),
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
										ClaimName: configVolumeClaim.Metadata.Name().Elem(),
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("home-assistant-config-map-mount"),
									ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
										Name:  configMap.Metadata.Name().Elem(),
										Items: generateConfigMapItems(configMapData),
									},
								},
								&corev1.VolumeArgs{
									Name: pulumi.String("home-assistant-secrets-mount"),
									Secret: &corev1.SecretVolumeSourceArgs{
										SecretName: secrets.Metadata.Name().Elem(),
										Items: &corev1.KeyToPathArray{
											&corev1.KeyToPathArgs{
												Key:  pulumi.String("LUTRON_CASETA_CA_CERT"),
												Path: pulumi.String("ca.crt"),
											},
											&corev1.KeyToPathArgs{
												Key:  pulumi.String("LUTRON_CASETA_CLIENT_CERT"),
												Path: pulumi.String("client.crt"),
											},
											&corev1.KeyToPathArgs{
												Key:  pulumi.String("LUTRON_CASETA_CLIENT_KEY"),
												Path: pulumi.String("client.key"),
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
				configVolumeClaim,
			}),
		); err != nil {
			return err
		}

		// var service *corev1.Service
		if _, err = corev1.NewService(
			ctx,
			"home-assistant-service",
			&corev1.ServiceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Namespace: namespace.Metadata.Name(),
					Name:      pulumi.String("home-assistant"),
				},
				Spec: &corev1.ServiceSpecArgs{
					Type:            pulumi.String("LoadBalancer"),
					SessionAffinity: pulumi.String("None"),
					Selector:        deploymentLabels,
					Ports: &corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:       pulumi.String("http"),
							Port:       pulumi.Int(8123),
							TargetPort: pulumi.String("http"),
							Protocol:   pulumi.String("TCP"),
						},
					},
				},
			},
			pulumi.Parent(deployment),
		); err != nil {
			return nil
		}

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
