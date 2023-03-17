package components

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type Component interface {
	Provision(ctx *pulumi.Context, name string) ([]pulumi.Resource, error)
}
