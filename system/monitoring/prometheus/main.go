package main

import (
	"github.com/kurtschwarz/home/system/monitoring/prometheus/resources"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "prometheus")
		namespace := config.Require("namespace")

		var operator *resources.PrometheusOperator
		if operator, err = resources.NewPrometheusOperator(
			ctx,
			"prometheus-operator",
			&resources.PrometheusOperatorArgs{
				Version:   config.Require("operatorVersion"),
				Namespace: namespace,
			},
		); err != nil {
			return err
		}

		if _, err = resources.NewPrometheus(
			ctx,
			"prometheus",
			&resources.PrometheusArgs{
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
