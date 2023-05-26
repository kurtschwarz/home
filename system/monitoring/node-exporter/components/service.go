package components

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiext "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NodeExporterService struct {
	pulumi.ResourceState

	DaemonSet *appsv1.DaemonSet

	Service            *corev1.Service
	ServiceAccount     *corev1.ServiceAccount
	ServiceRole        *rbacv1.ClusterRole
	ServiceRoleBinding *rbacv1.ClusterRoleBinding
	ServiceMonitor     *apiext.CustomResource
}

type NodeExporterServiceArgs struct {
	Namespace pulumi.StringInput
	Image     pulumi.StringInput
}

func NewNodeExporterService(
	ctx *pulumi.Context, name string, args *NodeExporterServiceArgs, opts ...pulumi.ResourceOption,
) (component *NodeExporterService, err error) {
	component = &NodeExporterService{}

	if err = ctx.RegisterComponentResource(
		"kurtschwarz:home/system/monitoring/node-exporter:Service",
		name,
		component,
		opts...,
	); err != nil {
		return nil, err
	}

	if component.ServiceAccount, err = corev1.NewServiceAccount(
		ctx,
		fmt.Sprintf("%s-service-account", name),
		&corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("node-exporter"),
				Namespace: args.Namespace,
			},
		},
		pulumi.Parent(component),
	); err != nil {
		return nil, err
	}

	if component.ServiceRole, err = rbacv1.NewClusterRole(
		ctx,
		fmt.Sprintf("%s-service-role", name),
		&rbacv1.ClusterRoleArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("node-exporter"),
			},
			Rules: &rbacv1.PolicyRuleArray{
				&rbacv1.PolicyRuleArgs{
					ApiGroups: &pulumi.StringArray{
						pulumi.String("authentication.k8s.io"),
					},
					Resources: &pulumi.StringArray{
						pulumi.String("tokenreviews"),
					},
					Verbs: &pulumi.StringArray{
						pulumi.String("create"),
					},
				},
				&rbacv1.PolicyRuleArgs{
					ApiGroups: &pulumi.StringArray{
						pulumi.String("authorization.k8s.io"),
					},
					Resources: &pulumi.StringArray{
						pulumi.String("subjectaccessreviews"),
					},
					Verbs: &pulumi.StringArray{
						pulumi.String("create"),
					},
				},
			},
		},
		pulumi.Parent(component.ServiceAccount),
	); err != nil {
		return nil, err
	}

	if component.ServiceRoleBinding, err = rbacv1.NewClusterRoleBinding(
		ctx,
		fmt.Sprintf("%s-service-role-binding", name),
		&rbacv1.ClusterRoleBindingArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("node-exporter"),
			},
			RoleRef: &rbacv1.RoleRefArgs{
				ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
				Kind:     pulumi.String("ClusterRole"),
				Name:     pulumi.String("node-exporter"),
			},
			Subjects: &rbacv1.SubjectArray{
				&rbacv1.SubjectArgs{
					Kind:      pulumi.String("ServiceAccount"),
					Name:      pulumi.String("node-exporter"),
					Namespace: args.Namespace,
				},
			},
		},
		pulumi.Parent(component.ServiceRole),
	); err != nil {
		return nil, err
	}

	labels := &pulumi.StringMap{
		"app.kubernetes.io/component": pulumi.String("exporter"),
		"app.kubernetes.io/name":      pulumi.String("node-exporter"),
	}

	if component.DaemonSet, err = appsv1.NewDaemonSet(
		ctx,
		fmt.Sprintf("%s-daemon-set", name),
		&appsv1.DaemonSetArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Labels:    labels,
				Name:      pulumi.String("node-exporter"),
				Namespace: args.Namespace,
			},
			Spec: &appsv1.DaemonSetSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: labels,
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: labels,
					},
					Spec: &corev1.PodSpecArgs{
						HostNetwork: pulumi.Bool(true),
						HostPID:     pulumi.Bool(true),
						SecurityContext: &corev1.PodSecurityContextArgs{
							RunAsNonRoot: pulumi.Bool(true),
							RunAsUser:    pulumi.Int(65534),
						},
						ServiceAccountName: component.ServiceAccount.Metadata.Name(),
						Containers: &corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String("node-exporter"),
								Image: args.Image,
								Args: pulumi.StringArray{
									pulumi.String("--path.procfs=/host/proc"),
									pulumi.String("--path.sysfs=/host/sys"),
									pulumi.String("--path.rootfs=/host/root"),
									pulumi.String("--collector.filesystem.ignored-mount-points=^/(dev|proc|sys|var/lib/docker/.+)($|/)"),
									pulumi.String("--collector.filesystem.ignored-fs-types=^(autofs|binfmt_misc|cgroup|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|mqueue|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|sysfs|tracefs)$"),
								},
								VolumeMounts: &corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										MountPath:        pulumi.String("/host/proc"),
										MountPropagation: pulumi.String("HostToContainer"),
										Name:             pulumi.String("proc"),
										ReadOnly:         pulumi.Bool(true),
									},
									&corev1.VolumeMountArgs{
										MountPath:        pulumi.String("/host/sys"),
										MountPropagation: pulumi.String("HostToContainer"),
										Name:             pulumi.String("sys"),
										ReadOnly:         pulumi.Bool(true),
									},
									&corev1.VolumeMountArgs{
										MountPath:        pulumi.String("/host/root"),
										MountPropagation: pulumi.String("HostToContainer"),
										Name:             pulumi.String("root"),
										ReadOnly:         pulumi.Bool(true),
									},
								},
								Ports: &corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(9100),
										HostPort:      pulumi.Int(9100),
										Name:          pulumi.String("node-exporter"),
									},
								},
							},
						},
						Volumes: &corev1.VolumeArray{
							&corev1.VolumeArgs{
								Name: pulumi.String("proc"),
								HostPath: &corev1.HostPathVolumeSourceArgs{
									Path: pulumi.String("/proc"),
								},
							},
							&corev1.VolumeArgs{
								Name: pulumi.String("sys"),
								HostPath: &corev1.HostPathVolumeSourceArgs{
									Path: pulumi.String("/sys"),
								},
							},
							&corev1.VolumeArgs{
								Name: pulumi.String("root"),
								HostPath: &corev1.HostPathVolumeSourceArgs{
									Path: pulumi.String("/"),
								},
							},
						},
					},
				},
			},
		},
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{
			component.ServiceAccount,
		}),
	); err != nil {
		return nil, err
	}

	if component.Service, err = corev1.NewService(
		ctx,
		fmt.Sprintf("%s-service", name),
		&corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Labels:    labels,
				Name:      pulumi.String("node-exporter"),
				Namespace: args.Namespace,
			},
			Spec: &corev1.ServiceSpecArgs{
				Ports: &corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("node-exporter"),
						Protocol:   pulumi.String("TCP"),
						Port:       pulumi.Int(9100),
						TargetPort: pulumi.Int(9100),
					},
				},
				Selector: labels,
			},
		},
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{
			component.DaemonSet,
		}),
	); err != nil {
		return nil, err
	}

	if component.ServiceMonitor, err = apiext.NewCustomResource(
		ctx,
		fmt.Sprintf("%s-service-monitor", name),
		&apiext.CustomResourceArgs{
			ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
			Kind:       pulumi.String("ServiceMonitor"),
			Metadata: &metav1.ObjectMetaArgs{
				Labels:    labels,
				Name:      pulumi.String("node-exporter"),
				Namespace: args.Namespace,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": kubernetes.UntypedArgs{
					"endpoints": []kubernetes.UntypedArgs{
						{
							"interval": pulumi.String("30s"),
							"port":     pulumi.String("node-exporter"),
						},
					},
					"selector": kubernetes.UntypedArgs{
						"matchLabels": labels,
					},
				},
			},
		},
		pulumi.Parent(component.Service),
	); err != nil {
		return nil, err
	}

	return component, nil
}
