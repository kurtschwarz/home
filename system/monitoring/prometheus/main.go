package main

import (
	"github.com/kurtschwarz/home/system/monitoring/prometheus/components"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "prometheus")
		namespace := config.Require("namespace")

		var operator *components.PrometheusOperator
		if operator, err = components.NewPrometheusOperator(
			ctx,
			"prometheus-operator",
			&components.PrometheusOperatorArgs{
				Version:   config.Require("operatorVersion"),
				Namespace: namespace,
			},
		); err != nil {
			return err
		}

		if _, err = components.NewPrometheus(
			ctx,
			"prometheus",
			&components.PrometheusArgs{
				Namespace: pulumi.String(namespace),
			},
			pulumi.Parent(operator),
		); err != nil {
			return err
		}

		ctx.Export("namespace", pulumi.String(namespace))

		return nil
	})
}
