import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'
import { stringify } from 'yaml'

import * as helpers from '../../packages/helpers'
import { Ingress, IngressType } from '../../resources/ingress'

interface Output {
  namespace: pulumi.Output<string>
}

export = async function (): Promise<Output> {
  const config = new pulumi.Config('overseerr')
  const namespace = new kube.core.v1.Namespace('overseerr-namespace', {
    metadata: {
      name: 'overseerr',
    },
  })

  const configVolume = new helpers.LonghornVolume('overseerr-config', {
    size: '30Gi',
    accessMode: 'ReadWriteOnce',
    namespace: namespace.metadata.name,
    replicas: '3',
  })

  const labels = {
    app: 'overseerr'
  }

  const deployment = new kube.apps.v1.Deployment(
    'overseerr-deployment',
    {
      metadata: {
        namespace: namespace.metadata.name,
        labels,
      },
      spec: {
        selector: {
          matchLabels: labels,
        },
        replicas: 1,
        template: {
          metadata: {
            namespace: namespace.metadata.name,
            labels
          },
          spec: {
            containers: [
              {
                name: 'overseerr',
                image: config.require('image'),
                env: [],
                ports: [
                  {
                    name: 'http',
                    protocol: 'TCP',
                    containerPort: config.requireNumber('port'),
                  },
                ],
                volumeMounts: [
                  {
                    name: 'overseerr-config-volume',
                    mountPath: '/config',
                  },
                ],
              },
            ],
            volumes: [
              {
                name: 'overseerr-config-volume',
                persistentVolumeClaim: {
                  claimName: configVolume.persistentVolumeClaim.metadata.name,
                },
              },
            ]
          },
        },
      },
    },
    {
      parent: namespace,
    },
  )

  const service = new kube.core.v1.Service(
    'overseerr-service',
    {
      metadata: {
        namespace: namespace.metadata.name,
      },
      spec: {
        type: 'LoadBalancer',
        sessionAffinity: 'None',
        selector: labels,
        ports: [
          {
            name: 'http',
            protocol: 'TCP',
            port: config.requireNumber('port'),
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

  new Ingress(
    'overseerr',
    {
      type: IngressType.HTTP,
      domain: config.require('domain'),
      serviceName: service.metadata.name,
      servicePort: config.requireNumber('port'),
      namespace: namespace.metadata.name
    },
    {
      parent: service,
    }
  )

  return {
    namespace: namespace.metadata.name,
  }
}
