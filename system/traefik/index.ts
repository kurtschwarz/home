import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

interface Output {
  namespace: pulumi.Output<string>
}

export = async function (): Promise<Output> {
  const config = new pulumi.Config('traefik')
  const namespace = new kube.core.v1.Namespace(
    'traefik-namespace',
    {
      metadata: {
        name: config.require('namespace'),
      },
    },
  )

  new kube.yaml.ConfigFile(
    'traefik-crd',
    {
      file: 'https://raw.githubusercontent.com/traefik/traefik/v2.9/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml',
    },
  )

  const serviceAccount = new kube.core.v1.ServiceAccount(
    'traefik-service-account',
    {
      metadata: {
        namespace: namespace.metadata.name,
      },
    }
  )

  const role = new kube.rbac.v1.ClusterRole(
    'traefik-cluster-role',
    {
      metadata: {
        namespace: namespace.metadata.name,
        name: 'traefik-role',
      },
      rules: [
        {
          apiGroups: [''],
          resources: ['services', 'endpoints', 'secrets'],
          verbs: ['get', 'list', 'watch']
        },
        {
          apiGroups: ['extensions', 'networking.k8s.io'],
          resources: ['ingresses', 'ingressclasses'],
          verbs: ['get', 'list', 'watch'],
        },
        {
          apiGroups: ['extensions', 'networking.k8s.io'],
          resources: ['ingresses/status'],
          verbs: ['update'],
        },
        {
          apiGroups: ['traefik.containo.us'],
          resources: ['ingressroutes', 'ingressroutetcps', 'ingressrouteudps', 'middlewares', 'middlewaretcps', 'tlsoptions', 'tlsstores', 'traefikservices', 'serverstransports'],
          verbs: ['get', 'list', 'watch'],
        },
      ],
    },
    {
      dependsOn: [
        serviceAccount
      ]
    }
  )

  new kube.rbac.v1.ClusterRoleBinding(
    'traefik-cluster-role-binding',
    {
      metadata: {
        namespace: namespace.metadata.name,
      },
      roleRef: {
        apiGroup: 'rbac.authorization.k8s.io',
        kind: 'ClusterRole',
        name: role.metadata.name,
      },
      subjects: [
        {
          kind: 'ServiceAccount',
          name: serviceAccount.metadata.name,
          namespace: namespace.metadata.name,
        }
      ],
    },
    {
      parent: role,
    },
  )

  const configMap = new kube.core.v1.ConfigMap(
    'traefik-config-map',
    {
      metadata: {
        namespace: namespace.metadata.name,
      },
      data: {
        'traefik.yaml': config.require('traefik.yaml'),
      },
    },
  )

  const labels = {
    app: 'traefik',
  }

  const deployment = new kube.apps.v1.Deployment(
    'traefik-deployment',
    {
      metadata: {
        namespace: namespace.metadata.name,
        name: 'traefik',
        labels: {
          ...labels,
          'kubernetes.io/cluster-service': 'true',
        },
      },
      spec: {
        replicas: config.requireNumber('replicas'),
        selector: {
          matchLabels: labels,
        },
        template: {
          metadata: {
            namespace: namespace.metadata.name,
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
            serviceAccountName: serviceAccount.metadata.name,
            containers: [
              {
                name: 'traefik',
                image: config.require('image'),
                args: ['--configFile=/etc/traefik/traefik.yaml'],
                ports: [
                  {
                    name: 'web',
                    containerPort: 80,
                  },
                  {
                    name: 'web-secure',
                    containerPort: 443,
                  },
                  {
                    name: 'dashboard',
                    containerPort: 8080,
                  },
                  {
                    name: 'plex',
                    protocol: 'TCP',
                    containerPort: 32400,
                  },
                  {
                    name: 'syncthing-tcp',
                    protocol: 'TCP',
                    containerPort: 22000,
                  },
                  {
                    name: 'syncthing-udp',
                    protocol: 'UDP',
                    containerPort: 22000,
                  },
                ],
                volumeMounts: [
                  {
                    name: 'traefik-config-yaml',
                    mountPath: '/etc/traefik/traefik.yaml',
                    subPath: 'traefik.yaml',
                  },
                ],
              }
            ],
            volumes: [
              {
                name: 'traefik-config-yaml',
                configMap: {
                  name: configMap.metadata.name,
                  items: [
                    {
                      key: 'traefik.yaml',
                      path: 'traefik.yaml',
                    },
                  ],
                },
              }
            ],
          },
        },
      },
    },
    {
      dependsOn: [
        configMap,
      ],
    },
  )

  new kube.core.v1.Service(
    'traefik-service',
    {
      metadata: {
        namespace: namespace.metadata.name,
        name: 'traefik',
      },
      spec: {
        type: 'LoadBalancer',
        loadBalancerIP: config.require('loadBalancerIP'),
        externalTrafficPolicy: 'Local',
        internalTrafficPolicy: 'Cluster',
        ports: [
          {
            port: 80,
            name: 'web',
            targetPort: 'web',
          },
          {
            port: 443,
            name: 'web-secure',
            targetPort: 'web-secure',
          },
          {
            port: 32400,
            protocol: 'TCP',
            name: 'plex',
            targetPort: 'plex',
          },
          {
            name: 'syncthing-tcp',
            protocol: 'TCP',
            port: 22000,
            targetPort: 'syncthing-tcp',
          },
          {
            name: 'syncthing-udp',
            protocol: 'UDP',
            port: 22000,
            targetPort: 'syncthing-udp',
          },
        ],
        selector: labels,
      },
    },
    {
      dependsOn: [
        deployment,
      ],
    }
  )

  return {
    namespace: namespace.metadata.name,
  }
}
