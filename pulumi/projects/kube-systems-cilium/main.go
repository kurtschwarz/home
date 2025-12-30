package main

import (
	kube "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func buildChartValuesMap(m map[string]interface{}) pulumi.Map {
	result := pulumi.Map{}
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = buildChartValuesMap(val)
		case string:
			result[k] = pulumi.String(val)
		case bool:
			result[k] = pulumi.Bool(val)
		case float64:
			result[k] = pulumi.Float64(val)
		case int:
			result[k] = pulumi.Int(val)
		default:
			result[k] = pulumi.Sprintf("%v", val)
		}
	}

	return result
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		provider, err := kube.NewProvider(ctx, "homelab", &kube.ProviderArgs{
			Kubeconfig: pulumi.String("/home/pulumi/.kube/config"),
			Context:    pulumi.String("homelab"),
		})

		if err != nil {
			return err
		}

		conf := config.New(ctx, "cilium")

		chartValues := map[string]interface{}{}
		conf.RequireObject("chartValues", &chartValues)

		helm.NewChart(
			ctx,
			"cilium",
			&helm.ChartArgs{
				Chart:     pulumi.String("cilium"),
				Version:   pulumi.String(conf.Require("chartVersion")),
				Namespace: pulumi.String("kube-system"),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://helm.cilium.io"),
				},
				Values: buildChartValuesMap(chartValues),
			},
			pulumi.Provider(provider),
		)

		return nil
	})
}
