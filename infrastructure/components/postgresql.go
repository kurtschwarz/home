package components

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PostgreSQL struct {
	Image           string
	DefaultUser     *string
	DefaultPassword *string
}

func (c *PostgreSQL) provisionVolumes(ctx *pulumi.Context, name string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	persistentVolumeName := fmt.Sprintf("%s-pv", name)
	persistentVolumeLabels := pulumi.StringMap{"app": pulumi.String(name), "type": pulumi.String("local")}
	persistentVolume, err := corev1.NewPersistentVolume(ctx, persistentVolumeName, &corev1.PersistentVolumeArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(persistentVolumeName),
			Labels: persistentVolumeLabels,
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
				Path: pulumi.String("/mnt/appdata/postgres/data"),
			},
		},
	})

	if err != nil {
		return nil, nil, err
	}

	persistentVolumeClaimName := fmt.Sprintf("%s-pv-claim", name)
	persistentVolumeClaimLabels := pulumi.StringMap{"app": pulumi.String(name)}
	persistentVolumeClaim, err := corev1.NewPersistentVolumeClaim(ctx, persistentVolumeClaimName, &corev1.PersistentVolumeClaimArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(persistentVolumeClaimName),
			Labels: persistentVolumeClaimLabels,
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
	}, pulumi.DependsOn([]pulumi.Resource{persistentVolume}))

	if err != nil {
		return nil, nil, err
	}

	return persistentVolume, persistentVolumeClaim, nil
}

func (c *PostgreSQL) Provision(ctx *pulumi.Context, name string) ([]pulumi.Resource, error) {
	persistentVolume, persistentVolumeClaim, err := c.provisionVolumes(ctx, name)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	configMapName := fmt.Sprintf("%s-config-map", name)
	configMapLabels := pulumi.StringMap{"app": pulumi.String(name)}
	configMap, err := corev1.NewConfigMap(ctx, configMapName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(configMapName),
			Labels: configMapLabels,
		},
		Data: &pulumi.StringMap{
			// "POSTGRES_USER":     c.DefaultUser.ToStringPtrOutput().Elem(),
			// "POSTGRES_PASSWORD": c.DefaultPassword.ToStringPtrOutput().Elem(),
		},
	})

	if err != nil {
		return []pulumi.Resource{}, err
	}

	statefulSetName := fmt.Sprintf("%s-stateful-set", name)
	statefulSetLabels := pulumi.StringMap{"app": pulumi.String(name)}
	statefulSet, err := appsv1.NewStatefulSet(ctx, statefulSetName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(statefulSetName),
			Labels: statefulSetLabels,
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			ServiceName: pulumi.String(name),
			Replicas:    pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: statefulSetLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: statefulSetLabels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String(name),
							Image: pulumi.String(c.Image),
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String(name),
									ContainerPort: pulumi.Int(5432),
								},
							},
							EnvFrom: &corev1.EnvFromSourceArray{
								&corev1.EnvFromSourceArgs{
									ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
										Name: configMap.Metadata.Name(),
									},
								},
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.Sprintf("%s-pv-data", name),
									MountPath: pulumi.String("/var/lib/postgresql/data"),
								},
							},
						},
					},
					Volumes: &corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String(fmt.Sprintf("%s-pv-data", name)),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
								ClaimName: persistentVolumeClaim.Metadata.Name().Elem(),
							},
						},
					},
				},
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{
		persistentVolume,
		configMap,
	}))

	if err != nil {
		return []pulumi.Resource{}, err
	}

	serviceName := fmt.Sprintf("%s-service", name)
	serviceLabels := pulumi.StringMap{"app": pulumi.String(name)}
	service, err := corev1.NewService(ctx, serviceName, &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name:   pulumi.String(serviceName),
			Labels: serviceLabels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("NodePort"),
			Selector: &pulumi.StringMap{
				"app": pulumi.String(name),
			},
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port: pulumi.Int(5432),
					Name: pulumi.String(name),
				},
			},
		},
	})

	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{
		persistentVolume,
		persistentVolumeClaim,
		configMap,
		statefulSet,
		service,
	}, nil
}
