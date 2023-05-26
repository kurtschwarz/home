package resources

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PrometheusOperatorArgs struct {
	Namespace string
	Version   string
}

type PrometheusOperator struct {
	pulumi.ResourceState
	Bundle *yaml.ConfigFile
}

func NewPrometheusOperator(
	ctx *pulumi.Context, name string, args *PrometheusOperatorArgs, opts ...pulumi.ResourceOption,
) (operator *PrometheusOperator, err error) {
	operator = &PrometheusOperator{}

	if err = ctx.RegisterComponentResource(
		"kurtschwarz:home/system/prometheus:Operator",
		name,
		operator,
		opts...,
	); err != nil {
		return nil, err
	}

	if operator.Bundle, err = yaml.NewConfigFile(
		ctx,
		"prometheus-operator",
		&yaml.ConfigFileArgs{
			File: fmt.Sprintf("https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v%s/bundle.yaml", args.Version),
			Transformations: []yaml.Transformation{
				// change all default namespaces to the monitoring namespace
				func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
					if _, ok := state["metadata"]; ok {
						if _, ok := state["metadata"].(map[string]interface{})["namespace"]; ok {
							state["metadata"].(map[string]interface{})["namespace"] = args.Namespace
						}
					}
				},
				// change the default namespace in the ClusterRoleBinding to the monitoring namespace
				func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
					if state["kind"] == "ClusterRoleBinding" {
						state["subjects"].([]interface{})[0].(map[string]interface{})["namespace"] = args.Namespace
					}
				},
			},
		},
		pulumi.Parent(operator),
	); err != nil {
		return nil, err
	}

	return operator, nil
}
