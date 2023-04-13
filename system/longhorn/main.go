package main

import (
	"fmt"

	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		config := config.New(ctx, "longhorn")
		_, err := yaml.NewConfigFile(
			ctx,
			"longhorn-manifest",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://raw.githubusercontent.com/longhorn/longhorn/v%s/deploy/longhorn.yaml", config.Require("version")),
			},
		)

		return err
	})
}
