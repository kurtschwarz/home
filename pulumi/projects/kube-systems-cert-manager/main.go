package main

import (
	kube "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	core "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
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

		conf := config.New(ctx, "certManager")

		chartValues := map[string]interface{}{}
		conf.RequireObject("chartValues", &chartValues)

		namespace, err := core.NewNamespace(
			ctx,
			"cert-manager",
			&core.NamespaceArgs{
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("cert-manager"),
				},
			},
			pulumi.Provider(provider),
		)

		if err != nil {
			return err
		}

		chart, err := helm.NewRelease(
			ctx,
			"cert-manager",
			&helm.ReleaseArgs{
				Name:      pulumi.String("cert-manager"),
				Chart:     pulumi.String("cert-manager"),
				Version:   pulumi.String(conf.Require("chartVersion")),
				Namespace: namespace.Metadata.Name(),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://charts.jetstack.io"),
				},
				Values: buildChartValuesMap(chartValues),
				// roll back automatically if an upgrade fails to become ready
				Atomic:        pulumi.Bool(true),
				CleanupOnFail: pulumi.Bool(true),
			},
			pulumi.Provider(provider),
		)

		if err != nil {
			return err
		}

		// Cloudflare API token for DNS-01 ACME challenges, referenced by the
		// ClusterIssuer below.
		cloudflareToken, err := core.NewSecret(
			ctx,
			"cloudflare-api-token",
			&core.SecretArgs{
				Metadata: &meta.ObjectMetaArgs{
					Name:      pulumi.String("cloudflare-api-token"),
					Namespace: namespace.Metadata.Name(),
				},
				StringData: pulumi.StringMap{
					"api-token": conf.RequireSecret("cloudflareApiToken"),
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart}),
		)

		if err != nil {
			return err
		}

		// Let's Encrypt (production) via DNS-01 against Cloudflare. Referenced
		// by Gateway/Ingress annotations as `cert-manager.io/cluster-issuer`.
		if _, err = apiextensions.NewCustomResource(
			ctx,
			"letsencrypt-prod",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cert-manager.io/v1"),
				Kind:       pulumi.String("ClusterIssuer"),
				Metadata: &meta.ObjectMetaArgs{
					Name: pulumi.String("letsencrypt-prod"),
				},
				OtherFields: kube.UntypedArgs{
					"spec": pulumi.Map{
						"acme": pulumi.Map{
							"email":  pulumi.String(conf.Require("letsEncryptEmail")),
							"server": pulumi.String("https://acme-v02.api.letsencrypt.org/directory"),
							"privateKeySecretRef": pulumi.Map{
								"name": pulumi.String("letsencrypt-prod-account-key"),
							},
							"solvers": pulumi.Array{
								pulumi.Map{
									"dns01": pulumi.Map{
										"cloudflare": pulumi.Map{
											"apiTokenSecretRef": pulumi.Map{
												"name": cloudflareToken.Metadata.Name(),
												"key":  pulumi.String("api-token"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{chart, cloudflareToken}),
		); err != nil {
			return err
		}

		return nil
	})
}
