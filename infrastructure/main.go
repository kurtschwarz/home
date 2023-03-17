package main

import (
	"infrastructure/components"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ProvisionComponent[C components.Component](ctx *pulumi.Context, name string, component C) ([]pulumi.Resource, error) {
	return component.Provision(ctx, name)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ProvisionComponent(ctx, "postgresql", &components.PostgreSQL{
			Image: "docker.io/library/postgres:15.2@sha256:50a96a21f2992518c2cb4601467cf27c7ac852542d8913c1872fe45cd6449947",
		})

		return nil
	})
}
