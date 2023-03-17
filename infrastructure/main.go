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
		var postgres components.Postgres
		var redis components.Redis

		config := config.New(ctx, "")
		config.RequireSecretObject("postgres", &postgres)
		config.RequireSecretObject("redis", &redis)

		ProvisionComponent(ctx, "postgres", &postgres)
		ProvisionComponent(ctx, "redis", &redis)

		return nil
	})
}
