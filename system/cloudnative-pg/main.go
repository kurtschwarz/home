package main

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "cnpg")

		var manifest *yaml.ConfigFile
		if manifest, err = yaml.NewConfigFile(
			ctx,
			"cnpg-manifest",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/v%[1]s/releases/cnpg-%[1]s.yaml", config.Require("version")),
			},
		); err != nil {
			return err
		}

		var namespace = manifest.GetResource("v1/Namespace", "cnpg-system", "").(*corev1.Namespace)

		ctx.Export("namespace", namespace.Metadata.Name())
		ctx.Export("version", pulumi.String(config.Require("version")))

		return nil
	})
}
