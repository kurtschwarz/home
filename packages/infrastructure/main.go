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

func MergeVolumeMountArray(mounts ...*corev1.VolumeMountArray) *corev1.VolumeMountArray {
	merged := corev1.VolumeMountArray{}

	for _, mount := range mounts {
		merged = append(merged, *mount...)
	}

	return &merged
}

func StringMapToEnvArray(m map[string]string) corev1.EnvVarArray {
	envs := make(corev1.EnvVarArray, len(m))

	for key, value := range m {
		envs = append(envs, &corev1.EnvVarArgs{
			Name:  pulumi.String(key),
			Value: pulumi.String(value),
		})
	}

	return envs
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
