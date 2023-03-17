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
		addresses := []string{}

		config := config.New(ctx, "metalLb")
		config.RequireObject("addresses", &addresses)

		_, err := apiextensions.NewCustomResource(ctx, "metal-lb-ip-address-pool", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("metallb.io/v1beta1"),
			Kind:       pulumi.String("IPAddressPool"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("custom-addresspool"),
				Namespace: pulumi.String("metallb-system"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"addresses": addresses,
				},
			},
		})

		return err
	})
}
