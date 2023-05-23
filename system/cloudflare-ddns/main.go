package main

import (
	"os"

	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "cloudflareDDNS")

		var namespace *corev1.Namespace
		if namespace, err = corev1.NewNamespace(
			ctx,
			"cloudflare-ddns-namespace",
			&corev1.NamespaceArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String(config.Require("namespace")),
				},
			},
		); err != nil {
			return err
		}

		var scriptBytes []byte
		if scriptBytes, err = os.ReadFile("ddns/ddns.go"); err != nil {
			return err
		}

		var configMap *corev1.ConfigMap
		if configMap, err = corev1.NewConfigMap(
			ctx,
			"cloudflare-ddns-config",
			&corev1.ConfigMapArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("cloudflare-ddns-config"),
					Namespace: namespace.Metadata.Name(),
				},
				Data: pulumi.StringMap{
					"ddns.go":   pulumi.String(scriptBytes),
					"ddns.json": pulumi.String(config.Require("ddns.json")),
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		var secrets *corev1.Secret
		if secrets, err = corev1.NewSecret(
			ctx,
			"cloudflare-ddns-secrets",
			&corev1.SecretArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("cloudflare-ddns-secrets"),
					Namespace: namespace.Metadata.Name(),
				},
				StringData: &pulumi.StringMap{
					"CF_API_TOKEN": config.RequireSecret("cloudflareAPIToken"),
				},
			},
			pulumi.Parent(namespace),
		); err != nil {
			return err
		}

		if _, err = batchv1.NewCronJob(
			ctx,
			"cloudflare-ddns-cron",
			&batchv1.CronJobArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("cloudflare-ddns-cron"),
					Namespace: namespace.Metadata.Name(),
				},
				Spec: &batchv1.CronJobSpecArgs{
					Schedule:                   pulumi.String(config.Require("schedule")),
					SuccessfulJobsHistoryLimit: pulumi.Int(10),
					FailedJobsHistoryLimit:     pulumi.Int(30),
					JobTemplate: &batchv1.JobTemplateSpecArgs{
						Spec: &batchv1.JobSpecArgs{
							Template: &corev1.PodTemplateSpecArgs{
								Spec: &corev1.PodSpecArgs{
									Containers: &corev1.ContainerArray{
										&corev1.ContainerArgs{
											Name:  pulumi.String("cloudflare-ddns"),
											Image: pulumi.String(config.Require("image")),
											Command: pulumi.StringArray{
												pulumi.String("go"),
												pulumi.String("run"),
												pulumi.String("/ddns.go"),
											},
											VolumeMounts: &corev1.VolumeMountArray{
												&corev1.VolumeMountArgs{
													Name:      pulumi.String("cloudflare-ddns-config-script"),
													MountPath: pulumi.String("/ddns.go"),
													SubPath:   pulumi.String("ddns.go"),
												},
												&corev1.VolumeMountArgs{
													Name:      pulumi.String("cloudflare-ddns-config-script"),
													MountPath: pulumi.String("/ddns.json"),
													SubPath:   pulumi.String("ddns.json"),
												},
											},
											EnvFrom: &corev1.EnvFromSourceArray{
												&corev1.EnvFromSourceArgs{
													SecretRef: &corev1.SecretEnvSourceArgs{
														Name:     secrets.Metadata.Name(),
														Optional: pulumi.Bool(false),
													},
												},
											},
										},
									},
									Volumes: &corev1.VolumeArray{
										&corev1.VolumeArgs{
											Name: pulumi.String("cloudflare-ddns-config-script"),
											ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
												Name: configMap.Metadata.Name(),
												Items: &corev1.KeyToPathArray{
													&corev1.KeyToPathArgs{
														Key:  pulumi.String("ddns.go"),
														Path: pulumi.String("ddns.go"),
													},
													&corev1.KeyToPathArgs{
														Key:  pulumi.String("ddns.json"),
														Path: pulumi.String("ddns.json"),
													},
												},
											},
										},
									},
									RestartPolicy: pulumi.String("OnFailure"),
								},
							},
						},
					},
				},
			},
			pulumi.Parent(namespace),
			pulumi.DependsOn([]pulumi.Resource{
				secrets,
				configMap,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
