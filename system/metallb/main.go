package main

import (
	"fmt"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		addresses := []string{}

		config := config.New(ctx, "metalLb")
		config.RequireObject("addresses", &addresses)

		var metallb *yaml.ConfigFile
		if metallb, err = yaml.NewConfigFile(
			ctx,
			"metallb-resources",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://raw.githubusercontent.com/metallb/metallb/%s/config/manifests/metallb-native.yaml", config.Require("version")),
			},
		); err != nil {
			return err
		}

		namespace := metallb.GetResource("v1/Namespace", "metallb-system", "").(*corev1.Namespace)

		var addressPool *apiextensions.CustomResource
		if addressPool, err = apiextensions.NewCustomResource(
			ctx,
			"metallb-ip-address-pool",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("metallb.io/v1beta1"),
				Kind:       pulumi.String("IPAddressPool"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("unifi-address-pool"),
					Namespace: addressPool.Metadata.Name(),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"protocol":  "layer2",
						"addresses": addresses,
					},
				},
			},
			pulumi.Parent(metallb),
			pulumi.DependsOn([]pulumi.Resource{
				namespace,
			}),
		); err != nil {
			return err
		}

		// var l2Advertisement *apiextensions.CustomResource
		if _, err = apiextensions.NewCustomResource(
			ctx,
			"metallb-unifi-l2-advertisement",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("metallb.io/v1beta1"),
				Kind:       pulumi.String("L2Advertisement"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("unifi-l2-advertisement"),
					Namespace: addressPool.Metadata.Name(),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"ipAddressPools": []string{
							"unifi-address-pool",
						},
					},
				},
			},
			pulumi.Parent(metallb),
			pulumi.DependsOn([]pulumi.Resource{
				namespace,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
