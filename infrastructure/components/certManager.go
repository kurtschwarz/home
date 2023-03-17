package components

import (
	"fmt"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CertManager struct {
	AcmeEmail string
}

func (c *CertManager) Provision(ctx *pulumi.Context, name string) ([]pulumi.Resource, error) {
	clusterIssuerName := fmt.Sprintf("%s-cluster-issuer", name)
	clusterIssuer, err := apiextensions.NewCustomResource(ctx, clusterIssuerName, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("ClusterIssuer"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("letsencrypt"),
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": kubernetes.UntypedArgs{
				"acme": kubernetes.UntypedArgs{
					"email": pulumi.String(c.AcmeEmail),
					"server": pulumi.String("https://acme-v02.api.letsencrypt.org/directory"),
					"privateKeySecretRef": kubernetes.UntypedArgs{
						"name": pulumi.String("letsencrypt-account-key"),
					},
					"solvers": []kubernetes.UntypedArgs{
						kubernetes.UntypedArgs{
							"http01": kubernetes.UntypedArgs{
								"ingress": kubernetes.UntypedArgs{
									"class": "public",
								},
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{clusterIssuer}, nil
}
