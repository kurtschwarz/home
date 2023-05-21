package infrastructure

import (
	"fmt"
	"strconv"

	humanize "github.com/dustin/go-humanize"
	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VolumeAccessMode string

const (
	ReadOnlyMany  VolumeAccessMode = "ReadOnlyMany"
	ReadWriteOnce VolumeAccessMode = "ReadWriteOnce"
	ReadWriteMany VolumeAccessMode = "ReadWriteMany"
)

var VolumeAccessModeLonghornMap = map[VolumeAccessMode]string{
	ReadOnlyMany:  "rox",
	ReadWriteOnce: "rwo",
	ReadWriteMany: "rwx",
}

type LonghornVolumeArgs struct {
	Size       pulumi.StringInput `pulumi:"size"`
	Replicas   pulumi.IntInput    `pulumi:"replicas"`
	Namespace  pulumi.StringInput `pulumi:"namespace"`
	AccessMode VolumeAccessMode   `pulumi:"accessMode"`
}

type LonghornVolume struct {
	pulumi.ResourceState

	LonghornNamespace pulumi.StringOutput
	LonghornVolume    *apiextensions.CustomResource

	PersistentVolume      *corev1.PersistentVolume
	PersistentVolumeClaim *corev1.PersistentVolumeClaim
}

func NewLonghornVolume(ctx *pulumi.Context, name string, args *LonghornVolumeArgs, opts ...pulumi.ResourceOption) (*LonghornVolume, error) {
	var resource = &LonghornVolume{}
	var err error

	if err = ctx.RegisterComponentResource(
		"kurtschwarz:home/packages/infrastructure:LonghornVolume",
		name,
		resource,
		opts...,
	); err != nil {
		return nil, err
	}

	resource.LonghornNamespace = RequireNamespace(ctx, "homelab-system-longhorn", ctx.Stack())

	// longhorn volume

	if resource.LonghornVolume, err = apiextensions.NewCustomResource(
		ctx,
		fmt.Sprintf("%s-longhorn-volume", name),
		&apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("longhorn.io/v1beta2"),
			Kind:       pulumi.String("Volume"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(name),
				Namespace: resource.LonghornNamespace,
				Labels: &pulumi.StringMap{
					"longhornvolume": pulumi.String(name),
				},
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"accessMode":       pulumi.String(VolumeAccessModeLonghornMap[args.AccessMode]),
					"frontend":         pulumi.String("blockdev"),
					"numberOfReplicas": args.Replicas,
					"size": args.Size.ToStringOutput().ApplyT(func(v string) pulumi.String {
						bytes, _ := humanize.ParseBytes(v)
						return pulumi.String(strconv.FormatUint(bytes, 10))
					}).(pulumi.StringInput),
				},
			},
		},
		pulumi.Parent(resource),
	); err != nil {
		return nil, err
	}

	// kube pv

	if resource.PersistentVolume, err = corev1.NewPersistentVolume(
		ctx,
		fmt.Sprintf("%s-pv", name),
		&corev1.PersistentVolumeArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.Sprintf("%s-pv", name),
				Namespace: args.Namespace,
			},
			Spec: &corev1.PersistentVolumeSpecArgs{
				Capacity: &pulumi.StringMap{
					"storage": args.Size,
				},
				Csi: &corev1.CSIPersistentVolumeSourceArgs{
					Driver:       pulumi.String("driver.longhorn.io"),
					FsType:       pulumi.String("ext4"),
					VolumeHandle: pulumi.String(name),
					VolumeAttributes: pulumi.StringMap{
						"numberOfReplicas": pulumi.Sprintf("%d", args.Replicas),
					},
				},
				AccessModes: &pulumi.StringArray{
					pulumi.String(args.AccessMode),
				},
				PersistentVolumeReclaimPolicy: pulumi.String("Retain"),
				StorageClassName:              pulumi.String("longhorn"),
				VolumeMode:                    pulumi.String("Filesystem"),
			},
		},
		pulumi.Parent(resource),
	); err != nil {
		return nil, err
	}

	// kube pvc

	if resource.PersistentVolumeClaim, err = corev1.NewPersistentVolumeClaim(
		ctx,
		fmt.Sprintf("%s-pvc", name),
		&corev1.PersistentVolumeClaimArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.Sprintf("%s-pvc", name),
				Namespace: args.Namespace,
				Annotations: &pulumi.StringMap{
					"pulumi.com/skipAwait": pulumi.String("true"),
				},
			},
			Spec: &corev1.PersistentVolumeClaimSpecArgs{
				AccessModes: &pulumi.StringArray{
					pulumi.String(args.AccessMode),
				},
				Resources: &corev1.ResourceRequirementsArgs{
					Requests: &pulumi.StringMap{
						"storage": args.Size,
					},
				},
				VolumeName:       resource.PersistentVolume.Metadata.Name(),
				StorageClassName: pulumi.String("longhorn"),
				VolumeMode:       pulumi.String("Filesystem"),
			},
		},
		pulumi.Parent(resource),
	); err != nil {
		return nil, err
	}

	return resource, nil
}

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
