package main

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "twingate")

		var namespace *corev1.Namespace
		namespace, err = corev1.NewNamespace(ctx, "twingate-namespace", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("twingate"),
			},
		})

		if err != nil {
			return err
		}

		connectors := make([]twingateConnectorArgs, 0)
		config.RequireObject("connectors", &connectors)

		for _, connector := range connectors {
			if _, err = NewTwingateConnector(
				ctx,
				fmt.Sprintf("twingate-connector-%s", connector.Name),
				TwingateConnectorArgs{
					Image:        pulumi.String(config.Require("image")),
					Namespace:    namespace.Metadata.Name(),
					TenantURL:    config.RequireSecret("tenantUrl"),
					AccessToken:  pulumi.String(connector.AccessToken),
					RefreshToken: pulumi.String(connector.RefreshToken),
				},
				pulumi.Parent(
					namespace,
				),
			); err != nil {
				return err
			}
		}

		return err
	})
}
