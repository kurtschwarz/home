package infrastructure

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func RequireNamespace(ctx *pulumi.Context, project string, env string) pulumi.StringOutput {
	ref, err := pulumi.NewStackReference(
		ctx,
		fmt.Sprintf("kurtschwarz/%s/%s", project, env),
		nil,
	)

	if err != nil {
		panic(err)
	}

	return ref.GetOutput(pulumi.String("namespace")).AsStringOutput()
}

func MergeStringMap(m ...pulumi.StringMap) pulumi.StringMap {
	o := pulumi.StringMap{}

	for i := range m {
		for k, v := range m[i] {
			o[k] = v
		}
	}

	return o
}
