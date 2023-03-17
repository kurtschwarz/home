package components

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Redis struct {
	Image    string
	Port     int
	Replicas int
}

func (c *Redis) Provision(ctx *pulumi.Context, name string) ([]pulumi.Resource, error) {
	statefulSetName := fmt.Sprintf("%s-stateful-set", name)
	statefulSetLabels := pulumi.StringMap{"app": pulumi.String(name)}
	statefulSet, err := appsv1.NewStatefulSet(ctx, statefulSetName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(statefulSetName),
			Labels: statefulSetLabels,
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			ServiceName: pulumi.String(name),
			Replicas:    pulumi.Int(c.Replicas),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: statefulSetLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: statefulSetLabels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String(name),
							Image: pulumi.String(c.Image),
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String(name),
									ContainerPort: pulumi.Int(c.Port),
								},
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return []pulumi.Resource{}, err
	}

	serviceName := fmt.Sprintf("%s-service", name)
	serviceLabels := pulumi.StringMap{"app": pulumi.String(name)}
	service, err := corev1.NewService(ctx, serviceName, &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name:   pulumi.String(serviceName),
			Labels: serviceLabels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("NodePort"),
			Selector: &pulumi.StringMap{
				"app": pulumi.String(name),
			},
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port: pulumi.Int(c.Port),
					Name: pulumi.String(name),
				},
			},
		},
	})

	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{
		statefulSet,
		service,
	}, nil
}
