package main

import (
	"infrastructure/components"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func ProvisionComponent[C components.Component](ctx *pulumi.Context, name string, component C) ([]pulumi.Resource, error) {
	return component.Provision(ctx, name)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "")

		ProvisionComponent(ctx, "postgresql", &components.PostgreSQL{
			Image: config.Require("postgresImage"),
		})

		ProvisionComponent(ctx, "redis", &components.Redis{
			Image: config.Require("redisImage"),
			Port:  config.RequireInt("redisPort"),
		})

		return nil
	})
}
