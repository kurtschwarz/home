package main

import (
	kube "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	apps "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	core "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	meta "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func provisionNamespace(
	ctx *pulumi.Context,
	provider *kube.Provider,
) (ns *core.Namespace, err error) {
	return core.NewNamespace(
		ctx,
		"beszel",
		&core.NamespaceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name: pulumi.String("beszel"),
			},
		},
		pulumi.Provider(provider),
	)
}

type HubConfig struct {
	Image string  `json:"image"`
	Port  float32 `json:"port"`
}

func provisionHub(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	provider *kube.Provider,
) (deployment *apps.Deployment, service *core.Service, err error) {
	var hubConfig HubConfig
	if err = config.New(ctx, "beszel").GetObject("hub", &hubConfig); err != nil {
		return nil, nil, err
	}

	if deployment, err = apps.NewDeployment(
		ctx,
		"beszel-hub",
		&apps.DeploymentArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("beszel-hub"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DeploymentSpecArgs{
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app": pulumi.String("beszel-hub"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app": pulumi.String("beszel-hub"),
						},
					},
					Spec: &core.PodSpecArgs{
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("beszel-hub"),
								Image:           pulumi.String(hubConfig.Image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(hubConfig.Port),
										HostPort:      pulumi.Int(hubConfig.Port),
									},
								},
								RestartPolicy: pulumi.String("Always"),
							},
						},
					},
				},
			},
		},
		pulumi.Provider(provider),
	); err != nil {
		return nil, nil, err
	}

	if service, err = core.NewService(
		ctx,
		"beszel-hub",
		&core.ServiceArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("beszel-hub"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &core.ServiceSpecArgs{
				Selector: &pulumi.StringMap{
					"app": pulumi.String("beszel-hub"),
				},
				Ports: &core.ServicePortArray{
					&core.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(hubConfig.Port),
						TargetPort: pulumi.Int(hubConfig.Port),
					},
				},
			},
		},
		pulumi.Provider(provider),
	); err != nil {
		return nil, nil, err
	}

	return deployment, service, nil
}

type AgentConfig struct {
	Image string  `json:"image"`
	Port  float32 `json:"port"`
	Key   string  `json:"key"`
	Token string  `json:"token"`
}

func provisionAgents(
	ctx *pulumi.Context,
	namespace *core.Namespace,
	provider *kube.Provider,
) (daemonset *apps.DaemonSet, err error) {
	var agentConfig AgentConfig
	if err = config.New(ctx, "beszel").GetObject("agent", &agentConfig); err != nil {
		return nil, err
	}

	return apps.NewDaemonSet(
		ctx,
		"beszel-agent",
		&apps.DaemonSetArgs{
			Metadata: &meta.ObjectMetaArgs{
				Name:      pulumi.String("beszel-agent"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: &apps.DaemonSetSpecArgs{
				Selector: &meta.LabelSelectorArgs{
					MatchLabels: &pulumi.StringMap{
						"app": pulumi.String("beszel-agent"),
					},
				},
				Template: &core.PodTemplateSpecArgs{
					Metadata: &meta.ObjectMetaArgs{
						Labels: &pulumi.StringMap{
							"app": pulumi.String("beszel-agent"),
						},
					},
					Spec: &core.PodSpecArgs{
						HostNetwork: pulumi.Bool(true),
						DnsPolicy:   pulumi.String("ClusterFirstWithHostNet"),
						Containers: &core.ContainerArray{
							&core.ContainerArgs{
								Name:            pulumi.String("beszel-agent"),
								Image:           pulumi.String(agentConfig.Image),
								ImagePullPolicy: pulumi.String("IfNotPresent"),
								Env: &core.EnvVarArray{
									&core.EnvVarArgs{
										Name:  pulumi.String("LISTEN"),
										Value: pulumi.Sprintf("%i", agentConfig.Port),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("TOKEN"),
										Value: pulumi.String(agentConfig.Token),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("KEY"),
										Value: pulumi.String(agentConfig.Key),
									},
									&core.EnvVarArgs{
										Name:  pulumi.String("HUB_URL"),
										Value: pulumi.String("http://beszel-hub.beszel.svc.cluster.local:8090"),
									},
								},
								Ports: &core.ContainerPortArray{
									&core.ContainerPortArgs{
										ContainerPort: pulumi.Int(agentConfig.Port),
										HostPort:      pulumi.Int(agentConfig.Port),
									},
								},
								VolumeMounts: &core.VolumeMountArray{
									&core.VolumeMountArgs{
										Name:             pulumi.String("host-root"),
										MountPath:        pulumi.String("/host/root"),
										MountPropagation: pulumi.String("HostToContainer"),
										ReadOnly:         pulumi.Bool(true),
									},
								},
								RestartPolicy: pulumi.String("Always"),
								SecurityContext: &core.SecurityContextArgs{
									Privileged: pulumi.Bool(true),
									RunAsUser:  pulumi.Int(0),
								},
							},
						},
						Volumes: &core.VolumeArray{
							core.VolumeArgs{
								Name: pulumi.String("host-root"),
								HostPath: &core.HostPathVolumeSourceArgs{
									Path: pulumi.String("/"),
								},
							},
						},
					},
				},
			},
		},
		pulumi.Provider(provider),
	)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		var provider *kube.Provider
		if provider, err = kube.NewProvider(
			ctx,
			"homelab",
			&kube.ProviderArgs{
				Kubeconfig: pulumi.String("/home/pulumi/.kube/config"),
				Context:    pulumi.String("homelab"),
			},
		); err != nil {
			return err
		}

		var namespace *core.Namespace
		if namespace, err = provisionNamespace(ctx, provider); err != nil {
			return err
		}

		if _, _, err = provisionHub(ctx, namespace, provider); err != nil {
			return err
		}

		if _, err = provisionAgents(ctx, namespace, provider); err != nil {
			return err
		}

		return nil
	})
}
