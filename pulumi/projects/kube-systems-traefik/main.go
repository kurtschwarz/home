package main

import (
	kube "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	core "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	helmv4 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	meta "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func buildChartValues(v interface{}) pulumi.Input {
	switch val := v.(type) {
	case map[string]interface{}:
		return buildChartValuesMap(val)
	case []interface{}:
		arr := pulumi.Array{}
		for _, item := range val {
			arr = append(arr, buildChartValues(item))
		}
		return arr
	case string:
		return pulumi.String(val)
	case bool:
		return pulumi.Bool(val)
	case float64:
		return pulumi.Float64(val)
	case int:
		return pulumi.Int(val)
	case nil:
		// YAML null: marshal through as-is so helm treats it as "unset"
		return nil
	default:
		return pulumi.Sprintf("%v", val)
	}
}

func buildChartValuesMap(m map[string]interface{}) pulumi.Map {
	result := pulumi.Map{}
	for k, v := range m {
		result[k] = buildChartValues(v)
	}

	return result
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		var provider *kube.Provider
		if provider, err = kube.NewProvider(ctx, "homelab", &kube.ProviderArgs{
			Kubeconfig: pulumi.String("/home/pulumi/.kube/config"),
			Context:    pulumi.String("homelab"),
		}); err != nil {
			return err
		}

		conf := config.New(ctx, "traefik")

		chartValues := map[string]interface{}{}
		conf.RequireObject("chartValues", &chartValues)

		namespace, err := core.NewNamespace(
			ctx,
			"traefik",
			&core.NamespaceArgs{
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("traefik"),
				},
			},
			pulumi.Provider(provider),
		)

		if err != nil {
			return err
		}

		// The main traefik chart bundles its own CRDs but not the Gateway API
		// ones. Install the traefik-crds chart with gatewayAPI enabled so the
		// Gateway/GatewayClass resources the main chart creates have their CRDs
		// present first. Rendered with Chart, not Release: the CRD manifests
		// exceed the 1MB limit of a helm release secret.
		crds, err := helmv4.NewChart(
			ctx,
			"traefik-crds",
			&helmv4.ChartArgs{
				Chart:     pulumi.String("traefik-crds"),
				Version:   pulumi.String(conf.Require("crdsChartVersion")),
				Namespace: namespace.Metadata.Name(),
				RepositoryOpts: &helmv4.RepositoryOptsArgs{
					Repo: pulumi.String("https://traefik.github.io/charts"),
				},
				Values: pulumi.Map{
					// The main chart ships the traefik.io CRDs already.
					"traefik":    pulumi.Bool(false),
					"gatewayAPI": pulumi.Bool(true),
				},
			},
			pulumi.Provider(provider),
		)

		if err != nil {
			return err
		}

		if _, err = helm.NewRelease(
			ctx,
			"traefik",
			&helm.ReleaseArgs{
				Name:      pulumi.String("traefik"),
				Chart:     pulumi.String("traefik"),
				Version:   pulumi.String(conf.Require("chartVersion")),
				Namespace: namespace.Metadata.Name(),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://traefik.github.io/charts"),
				},
				Values: buildChartValuesMap(chartValues),
				// roll back automatically if an upgrade fails to become ready
				Atomic:        pulumi.Bool(true),
				CleanupOnFail: pulumi.Bool(true),
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{crds}),
		); err != nil {
			return err
		}

		return nil
	})
}
