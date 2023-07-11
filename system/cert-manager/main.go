package main

import (
	"fmt"

	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "certManager")

		var manifests *yaml.ConfigFile
		if manifests, err = yaml.NewConfigFile(
			ctx,
			"cert-manager-manifest",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml", config.Require("version")),
			},
		); err != nil {
			return err
		}

		var namespace = manifests.GetResource("v1/Namespace", "cert-manager", "").(*corev1.Namespace)

		var cloudflareSecret *corev1.Secret
		if cloudflareSecret, err = corev1.NewSecret(
			ctx,
			"cert-manager-cloudflare-secrets",
			&corev1.SecretArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("cert-manager-cloudflare-secrets"),
					Namespace: namespace.Metadata.Name(),
				},
				StringData: pulumi.StringMap{
					"CLOUDFLARE_API_TOKEN": config.RequireSecret("cloudflareToken"),
				},
			},
			pulumi.Parent(manifests),
			pulumi.DependsOn([]pulumi.Resource{
				namespace,
			}),
		); err != nil {
			return err
		}

		if _, err = apiextensions.NewCustomResource(
			ctx,
			"cert-manager-lets-encrypt-issuer",
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("cert-manager.io/v1"),
				Kind:       pulumi.String("ClusterIssuer"),
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String("cert-manager-lets-encrypt-issuer"),
				},
				OtherFields: kubernetes.UntypedArgs{
					"spec": kubernetes.UntypedArgs{
						"acme": kubernetes.UntypedArgs{
							"email":  pulumi.String(config.Require("acmeEmail")),
							"server": pulumi.String("https://acme-v02.api.letsencrypt.org/directory"),
							"privateKeySecretRef": kubernetes.UntypedArgs{
								"name": pulumi.String("letsencrypt-account-key"),
							},
							"solvers": []kubernetes.UntypedArgs{
								{
									"selector": kubernetes.UntypedArgs{
										"dnsZones": pulumi.StringArray{
											pulumi.String("kurtina.ca"),
											pulumi.String("kurtflix.ca"),
										},
									},
									"dns01": kubernetes.UntypedArgs{
										"cnameStrategy": pulumi.String("Follow"),
										"cloudflare": kubernetes.UntypedArgs{
											"apiTokenSecretRef": kubernetes.UntypedArgs{
												"name": cloudflareSecret.Metadata.Name(),
												"key":  pulumi.String("CLOUDFLARE_API_TOKEN"),
											},
										},
									},
								},
								{
									"http01": kubernetes.UntypedArgs{
										"ingress": kubernetes.UntypedArgs{
											"class": "traefik-cert-manager",
											"ingressTemplate": kubernetes.UntypedArgs{
												"metadata": kubernetes.UntypedArgs{
													"annotations": kubernetes.UntypedArgs{
														"traefik.ingress.kubernetes.io/router.priority": "100",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			pulumi.Parent(manifests),
			pulumi.DependsOn([]pulumi.Resource{
				namespace,
				cloudflareSecret,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
