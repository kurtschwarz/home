package main

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type twingateConnectorArgs struct {
	Name         string
	AccessToken  string
	RefreshToken string
}

type TwingateConnectorArgs struct {
	Image        pulumi.StringInput    `pulumi:"image"`
	Namespace    pulumi.StringPtrInput `pulumi:"namespace"`
	TenantURL    pulumi.StringInput    `pulumi:"tenantUrl"`
	AccessToken  pulumi.StringInput    `pulumi:"accessToken"`
	RefreshToken pulumi.StringInput    `pulumi:"refreshToken"`
}

type TwingateConnector struct {
	pulumi.ResourceState
}

func NewTwingateConnector(ctx *pulumi.Context, name string, args TwingateConnectorArgs, opts ...pulumi.ResourceOption) (*TwingateConnector, error) {
	var connector = &TwingateConnector{}
	var err error

	if err = ctx.RegisterComponentResource(
		"kurtschwarz:system/twingate:TwingateConnector",
		name,
		connector,
		opts...,
	); err != nil {
		return nil, err
	}

	var secrets *corev1.Secret
	if secrets, err = corev1.NewSecret(
		ctx,
		fmt.Sprintf("%s-secrets", name),
		&corev1.SecretArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: args.Namespace,
				Name:      pulumi.Sprintf("%s-secrets", name),
			},
			Type: pulumi.String("Opaque"),
			StringData: &pulumi.StringMap{
				"TWINGATE_ACCESS_TOKEN":  args.AccessToken,
				"TWINGATE_REFRESH_TOKEN": args.RefreshToken,
			},
		},
		pulumi.Parent(connector),
	); err != nil {
		return nil, err
	}

	var config *corev1.ConfigMap
	if config, err = corev1.NewConfigMap(
		ctx,
		fmt.Sprintf("%s-config", name),
		&corev1.ConfigMapArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: args.Namespace,
				Name:      pulumi.Sprintf("%s-config", name),
			},
			Data: &pulumi.StringMap{
				"TWINGATE_URL": args.TenantURL,
				"LOG_LEVEL":    pulumi.String("6"),
			},
		},
		pulumi.Parent(connector),
	); err != nil {
		return nil, err
	}

	labels := pulumi.StringMap{
		"app": pulumi.String("twingate-connector"),
	}

	if _, err = appsv1.NewDeployment(
		ctx,
		fmt.Sprintf("%s-deployment", name),
		&appsv1.DeploymentArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: args.Namespace,
				Name:      pulumi.String(name),
				Labels:    labels,
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: labels,
				},
				Replicas: pulumi.Int(1),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Namespace: args.Namespace,
						Labels:    labels,
					},
					Spec: &corev1.PodSpecArgs{
						Tolerations: &corev1.TolerationArray{
							&corev1.TolerationArgs{
								Key:      pulumi.String("CriticalAddonsOnly"),
								Operator: pulumi.String("Exists"),
							},
							&corev1.TolerationArgs{
								Key:      pulumi.String("node-role.kubernetes.io/control-plane"),
								Operator: pulumi.String("Exists"),
								Effect:   pulumi.String("NoSchedule"),
							},
							&corev1.TolerationArgs{
								Key:      pulumi.String("node-role.kubernetes.io/master"),
								Operator: pulumi.String("Exists"),
								Effect:   pulumi.String("NoSchedule"),
							},
						},
						PriorityClassName: pulumi.String("system-cluster-critical"),
						NodeSelector: pulumi.StringMap{
							"node-role.kubernetes.io/master": pulumi.String("true"),
						},
						SecurityContext: &corev1.PodSecurityContextArgs{
							Sysctls: &corev1.SysctlArray{
								&corev1.SysctlArgs{
									Name:  pulumi.String("net.ipv4.ping_group_range"),
									Value: pulumi.String("0  2147483647"),
								},
							},
						},
						Containers: &corev1.ContainerArray{
							&corev1.ContainerArgs{
								Image: args.Image,
								Name:  pulumi.String(name),
								EnvFrom: &corev1.EnvFromSourceArray{
									&corev1.EnvFromSourceArgs{
										ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
											Name:     config.Metadata.Name(),
											Optional: pulumi.Bool(false),
										},
									},
									&corev1.EnvFromSourceArgs{
										SecretRef: &corev1.SecretEnvSourceArgs{
											Name:     secrets.Metadata.Name(),
											Optional: pulumi.Bool(false),
										},
									},
								},
								SecurityContext: &corev1.SecurityContextArgs{
									AllowPrivilegeEscalation: pulumi.Bool(false),
								},
							},
						},
					},
				},
			},
		},
		pulumi.Parent(connector),
		pulumi.DependsOn([]pulumi.Resource{
			secrets,
			config,
		}),
	); err != nil {
		return nil, err
	}

	return connector, nil
}
