package main

import (
	"fmt"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		addresses := []string{}

		config := config.New(ctx, "metalLb")
		config.RequireObject("addresses", &addresses)

		manifest, err := yaml.NewConfigFile(
			ctx,
			"metal-lb-manifest",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://raw.githubusercontent.com/metallb/metallb/%s/config/manifests/metallb-native.yaml", config.Require("version")),
			},
		)

		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(
			ctx,
			"metal-lb-ip-address-pool",
			&apiextensions.CustomResourceArgs{
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
			},
			pulumi.Parent(manifest),
		)

		return err
	})
}
