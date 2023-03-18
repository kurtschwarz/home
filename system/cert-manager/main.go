package main

import (
	"fmt"
	"strings"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "certManager")

		clusterIssuerName := "cert-manager-lets-encrypt-issuer"
		clusterIssuer, err := apiextensions.NewCustomResource(ctx, clusterIssuerName, &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("cert-manager.io/v1"),
			Kind:       pulumi.String("ClusterIssuer"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(clusterIssuerName),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"acme": kubernetes.UntypedArgs{
						"email":  pulumi.String(config.Require("acmeEmail")),
						"server": pulumi.String("https://acme-staging-v02.api.letsencrypt.org/directory"),
						"privateKeySecretRef": kubernetes.UntypedArgs{
							"name": pulumi.String("letsencrypt-account-key"),
						},
						"solvers": []kubernetes.UntypedArgs{
							{
								"http01": kubernetes.UntypedArgs{
									"ingress": kubernetes.UntypedArgs{
										"class": "traefik-cert-manager",
									},
								},
							},
						},
					},
				},
			},
		})

		domains := []string{}
		config.RequireObject("domains", &domains)

		for _, domain := range domains {
			domainName := strings.Replace(domain, ".", "-", -1)
			certificateName := fmt.Sprintf("cert-manager-cert-%s", domainName)
			_, err := apiextensions.NewCustomResource(ctx, certificateName, &apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cert-manager.io/v1"),
				Kind:       pulumi.String("Certificate"),
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String(certificateName),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"commonName": pulumi.String(domain),
						"secretName": pulumi.String(domainName),
						"dnsNames": pulumi.StringArray{
							pulumi.String(domain),
						},
						"issuerRef": kubernetes.UntypedArgs{
							"name": clusterIssuer.Metadata.Name().Elem(),
							"kind": "ClusterIssuer",
						},
					},
				},
			})

			if err != nil {
				return err
			}
		}

		return err
	})
}
