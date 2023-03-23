package infrastructure

import (
	"fmt"
	"strings"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func RequireNamespace(ctx *pulumi.Context, project string, env string) pulumi.StringOutput {
	ref, err := pulumi.NewStackReference(
		ctx,
		fmt.Sprintf("kurtschwarz/%s/%s", project, env),
		nil,
	)

	if err != nil {
		panic(err)
	}

	return ref.GetOutput(pulumi.String("namespace")).AsStringOutput()
}

func MergeStringMap(m ...pulumi.StringMap) pulumi.StringMap {
	o := pulumi.StringMap{}

	for i := range m {
		for k, v := range m[i] {
			o[k] = v
		}
	}

	return o
}

func ProvisionCertificate(ctx *pulumi.Context, namespace pulumi.StringInput, domain string) (*apiextensions.CustomResource, error) {
	name := strings.Replace(domain, ".", "-", -1)

	certificate, err := apiextensions.NewCustomResource(ctx, name, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("Certificate"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: namespace,
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": kubernetes.UntypedArgs{
				"commonName": pulumi.String(domain),
				"secretName": pulumi.String(name),
				"secretTemplate": kubernetes.UntypedArgs{
					"namespace": namespace,
				},
				"dnsNames": pulumi.StringArray{
					pulumi.String(domain),
				},
				"issuerRef": kubernetes.UntypedArgs{
					"name": pulumi.String("cert-manager-lets-encrypt-issuer"),
					"kind": pulumi.String("ClusterIssuer"),
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return certificate, nil
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
										Key:      pulumi.String("kubernetes.io/hostname"),
										Operator: pulumi.String("In"),
										Values: pulumi.StringArray{
											pulumi.String("kurtina-k8s-worker-unraid"),
										},
									},
								},
							},
						},
					},
				},
				StorageClassName:              pulumi.String("microk8s-hostpath"),
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
	); err != nil {
		return nil, nil, err
	}

	return persistentVolume, persistentVolumeClaim, nil
}
