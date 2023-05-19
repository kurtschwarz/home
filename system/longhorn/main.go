package main

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	config "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "longhorn")

		var longhorn *yaml.ConfigFile
		if longhorn, err = yaml.NewConfigFile(
			ctx,
			"longhorn-manifest",
			&yaml.ConfigFileArgs{
				File: fmt.Sprintf("https://raw.githubusercontent.com/longhorn/longhorn/v%s/deploy/longhorn.yaml", config.Require("version")),
			},
		); err != nil {
			return err
		}

		namespace := longhorn.GetResource("v1/Namespace", "longhorn-system", "").(*corev1.Namespace)

		s3CompatibleBackupSecretsMap := map[string]string{}
		config.RequireObject("s3CompatibleBackupSecrets", &s3CompatibleBackupSecretsMap)

		s3CompatibleBackupSecretsStringMap := &pulumi.StringMap{}
		for k, v := range s3CompatibleBackupSecretsMap {
			(*s3CompatibleBackupSecretsStringMap)[k] = pulumi.String(v)
		}

		if _, err = corev1.NewSecret(
			ctx,
			"longhorn-s3-compatible-backup-target-secret",
			&corev1.SecretArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("longhorn-s3-compatible-backup-target-secret"),
					Namespace: namespace.Metadata.Name(),
				},
				Type:       pulumi.String("Opaque"),
				StringData: s3CompatibleBackupSecretsStringMap,
			},
			pulumi.Parent(longhorn),
			pulumi.DependsOn([]pulumi.Resource{
				namespace,
			}),
		); err != nil {
			return err
		}

		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	})
}
