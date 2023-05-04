package infrastructure

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ProvisionLocalVolume(ctx *pulumi.Context, namespace pulumi.StringInput, name string, path string) (persistentVolume *corev1.PersistentVolume, persistentVolumeClaim *corev1.PersistentVolumeClaim, err error) {
	if persistentVolume, err = corev1.NewPersistentVolume(
		ctx,
		fmt.Sprintf("%s-pv", name),
		&corev1.PersistentVolumeArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.Sprintf("%s-pv", name),
				Labels: pulumi.StringMap{
					"type": pulumi.String("local"),
				},
				Namespace: namespace,
			},
			Spec: &corev1.PersistentVolumeSpecArgs{
				NodeAffinity: &corev1.VolumeNodeAffinityArgs{
					Required: &corev1.NodeSelectorArgs{
						NodeSelectorTerms: &corev1.NodeSelectorTermArray{
							&corev1.NodeSelectorTermArgs{
								MatchExpressions: &corev1.NodeSelectorRequirementArray{
									&corev1.NodeSelectorRequirementArgs{
										Key:      pulumi.String("disk"),
										Operator: pulumi.String("In"),
										Values: pulumi.StringArray{
											pulumi.String("unraid"),
										},
									},
								},
							},
						},
					},
				},
				StorageClassName:              pulumi.String("local-path"),
				PersistentVolumeReclaimPolicy: pulumi.String("Retain"),
				Capacity: pulumi.StringMap{
					"storage": pulumi.String("20Gi"),
				},
				AccessModes: &pulumi.StringArray{
					pulumi.String("ReadWriteOnce"),
				},
				HostPath: &corev1.HostPathVolumeSourceArgs{
					Path: pulumi.String(path),
				},
			},
		},
	); err != nil {
		return nil, nil, err
	}

	if persistentVolumeClaim, err = corev1.NewPersistentVolumeClaim(
		ctx,
		fmt.Sprintf("%s-pv-claim", name),
		&corev1.PersistentVolumeClaimArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.Sprintf("%s-pv-claim", name),
				Namespace: namespace,
				Annotations: &pulumi.StringMap{
					"pulumi.com/skipAwait": pulumi.String("true"),
				},
			},
			Spec: &corev1.PersistentVolumeClaimSpecArgs{
				StorageClassName: pulumi.String("local-path"),
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
	); err != nil {
		return nil, nil, err
	}

	return persistentVolume, persistentVolumeClaim, nil
}
