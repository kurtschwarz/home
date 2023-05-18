package main

import (
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func provisionCrdResources(ctx *pulumi.Context) (err error) {
	if _, err = yaml.NewConfigFile(
		ctx,
		"traefik-crd",
		&yaml.ConfigFileArgs{
			File: "https://raw.githubusercontent.com/traefik/traefik/v2.9/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml",
		},
	); err != nil {
		return err
	}

	return nil
}
