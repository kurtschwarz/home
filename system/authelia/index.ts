import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'
import { stringify } from 'yaml'

interface Output {
  namespace: pulumi.Output<string>
}

export = async function (): Promise<Output> {
  const config = new pulumi.Config('authelia')
  const namespace = new kube.core.v1.Namespace('authelia-namespace', {
    metadata: {
      name: 'authelia',
    },
  })

  const secrets = new kube.core.v1.Secret('authelia-secrets', {
    metadata: {
      namespace: namespace.metadata.name,
    },
    stringData: {
      'users_database.yml': config.requireSecretObject('users').apply(users => stringify({ users })),
    },
  }, {
    parent: namespace,
  })

  const configMap = new kube.core.v1.ConfigMap('authelia-config', {
    metadata: {
      namespace: namespace.metadata.name,
    },
    data: {
      'configuration.yml': config.requireSecretObject('config').apply(o => stringify(o)),
    },
  }, {
    parent: namespace,
  })

  const labels = { app: 'authelia' }
  const deployment = new kube.apps.v1.Deployment(
    'authelia-deployment',
    {
      metadata: {
        namespace: namespace.metadata.name,
        labels,
      },
      spec: {
        selector: {
          matchLabels: labels,
        },
        template: {
          metadata: {
            namespace: namespace.metadata.name,
            labels,
          },
          spec: {
            containers: [
              {
                name: 'authelia',
                image: config.require('image'),
                ports: [
                  {
                    name: 'http',
                    containerPort: 9091,
                  },
                ],
                volumeMounts: [
                  {
                    name: 'authelia-config',
                    mountPath: '/config/configuration.yml',
                    subPath: 'configuration.yml',
                  },
                  {
                    name: 'authelia-secrets',
                    mountPath: '/config/users_database.yml',
                    subPath: 'users_database.yml',
                  },
                ],
              },
            ],
            volumes: [
              {
                name: 'authelia-config',
                configMap: {
                  name: configMap.metadata.name,
                  items: [
                    {
                      key: 'configuration.yml',
                      path: 'configuration.yml',
                    },
                  ],
                },
              },
              {
                name: 'authelia-secrets',
                secret: {
                  secretName: secrets.metadata.name,
                  items: [
                    {
                      key: 'users_database.yml',
                      path: 'users_database.yml',
                    },
                  ],
                },
              },
            ],
          },
        },
      },
    },
    {
      parent: namespace,
      dependsOn: [
        configMap,
        secrets,
      ],
    },
  )

  new kube.core.v1.Service(
    'authelia-service',
    {
      metadata: {
        namespace: namespace.metadata.name,
      },
      spec: {
        selector: labels,
        ports: [
          {
            name: 'http',
            port: 9091,
            targetPort: 'http',
          },
        ],
      },
    },
    {
      parent: namespace,
      dependsOn: [
        deployment,
      ],
    },
  )

  return {
    namespace: namespace.metadata.name,
  }
}
