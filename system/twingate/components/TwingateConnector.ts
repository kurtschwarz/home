import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

interface TwingateConnectorArgs {
  image: pulumi.Input<string>
  namespace: pulumi.Input<string>
  tenantUrl: pulumi.Input<string>
  accessToken: pulumi.Input<string>
  refreshToken: pulumi.Input<string>
}

export class TwingateConnector extends pulumi.ComponentResource {
  private secrets: kube.core.v1.Secret
  private config: kube.core.v1.ConfigMap
  private deployment: kube.apps.v1.Deployment

  constructor(
    name: string,
    args: TwingateConnectorArgs,
    opts: pulumi.ComponentResourceOptions,
  ) {
    super('kurtschwarz:system/twingate:TwingateConnector', name, args, opts)

    this.secrets = new kube.core.v1.Secret(
      `${name}-secrets`,
      {
        metadata: {
          namespace: args.namespace,
          name: `${name}-secrets`,
        },
        type: 'Opaque',
        stringData: {
          TWINGATE_ACCESS_TOKEN: args.accessToken,
          TWINGATE_REFRESH_TOKEN: args.refreshToken,
        },
      },
      {
        parent: this,
      }
    )

    this.config = new kube.core.v1.ConfigMap(
      `${name}-config`,
      {
        metadata: {
          namespace: args.namespace,
          name: `${name}-config`,
        },
        data: {
          TWINGATE_URL: args.tenantUrl,
          DNS_SERVER: '10.35.5.5',
          LOG_LEVEL: '6',
        },
      },
      {
        parent: this,
      },
    )

    const labels = {
      app: 'twingate-connector'
    }

    this.deployment = new kube.apps.v1.Deployment(
      `${name}-deployment`,
      {
        metadata: {
          namespace: args.namespace,
          name,
          labels,
        },
        spec: {
          selector: {
            matchLabels: labels,
          },
          replicas: 1,
          template: {
            metadata: {
              namespace: args.namespace,
              labels,
            },
            spec: {
              tolerations: [
                {
                  key: 'CriticalAddonsOnly',
                  operator: 'Exists',
                },
                {
                  key: 'node-role.kubernetes.io/control-plane',
                  operator: 'Exists',
                  effect: 'NoSchedule',
                },
                {
                  key: 'node-role.kubernetes.io/master',
                  operator: 'Exists',
                  effect: 'NoSchedule',
                },
              ],
              priorityClassName: 'system-cluster-critical',
              nodeSelector: {
                'node-role.kubernetes.io/master': 'true',
              },
              securityContext: {
                sysctls: [
                  {
                    name: 'net.ipv4.ping_group_range',
                    value: '0  2147483647',
                  },
                ],
              },
              containers: [
                {
                  image: args.image,
                  name,
                  envFrom: [
                    {
                      configMapRef: {
                        name: this.config.metadata.name,
                        optional: false,
                      },
                    },
                    {
                      secretRef: {
                        name: this.secrets.metadata.name,
                        optional: false,
                      }
                    },
                  ],
                  securityContext: {
                    allowPrivilegeEscalation: false,
                  },
                },
              ],
            },
          },
        },
      },
      {
        parent: this,
        dependsOn: [
          this.secrets,
          this.config,
        ],
      }
    )
  }
}
