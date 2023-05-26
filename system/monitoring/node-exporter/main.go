package main

import (
	"github.com/kurtschwarz/home/system/monitoring/node-exporter/components"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "nodeExporter")
		namespace := config.Require("namespace")

		// var service *components.NodeExporterService
		if _, err = components.NewNodeExporterService(
			ctx,
			"node-exporter",
			&components.NodeExporterServiceArgs{
				Namespace: pulumi.String(namespace),
				Image:     pulumi.String(config.Require("image")),
			},
		); err != nil {
			return err
		}

		ctx.Export("namespace", pulumi.String(namespace))

		return nil
	})
}
