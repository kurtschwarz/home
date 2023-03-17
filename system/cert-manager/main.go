package main

import (
	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "certManager")
		_, err := apiextensions.NewCustomResource(ctx, "cert-manager-cluster-issuer", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("cert-manager.io/v1"),
			Kind:       pulumi.String("ClusterIssuer"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("letsencrypt"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"acme": kubernetes.UntypedArgs{
						"email":  pulumi.String(config.Require("acmeEmail")),
						"server": pulumi.String("https://acme-v02.api.letsencrypt.org/directory"),
						"privateKeySecretRef": kubernetes.UntypedArgs{
							"name": pulumi.String("letsencrypt-account-key"),
						},
						"solvers": []kubernetes.UntypedArgs{
							{
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

		return err
	})
}
