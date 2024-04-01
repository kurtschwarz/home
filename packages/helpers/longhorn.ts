import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

import { getNamespaceFromProject } from '.'

interface LonghornVolumeArgs {
  namespace: pulumi.Input<string>
  accessMode: pulumi.Input<string>
  size: pulumi.Input<string>
  replicas: pulumi.Input<string>
}

export class LonghornVolume extends pulumi.ComponentResource {
  public namespace: kube.core.v1.Namespace

  public persistentVolume: kube.core.v1.PersistentVolume
  public persistentVolumeClaim: kube.core.v1.PersistentVolumeClaim

  constructor(
    name: string,
    args: LonghornVolumeArgs,
    opts?: pulumi.ComponentResourceOptions
  ) {
    super('kurtschwarz:home/packages/helpers:LonghornVolume', name, args, opts)

    this.namespace = kube.core.v1.Namespace.get('longhorn-system', '', {})

    this.persistentVolume = new kube.core.v1.PersistentVolume(
      `${name}-pv`,
      {
        metadata: {
          name: `${name}-pv`,
          namespace: args.namespace,
        },
        spec: {
          capacity: {
            storage: args.size,
          },
          csi: {
            driver: 'driver.longhorn.io',
            fsType: 'ext4',
            volumeHandle: name,
            volumeAttributes: {
              numberOfReplicas: args.replicas,
            },
          },
          accessModes: [args.accessMode],
          persistentVolumeReclaimPolicy: 'Retain',
          storageClassName: 'longhorn',
          volumeMode: 'Filesystem',
        },
      },
      {
        parent: this,
      }
    )

    this.persistentVolumeClaim = new kube.core.v1.PersistentVolumeClaim(
      `${name}-pvc`,
      {
        metadata: {
          name: `${name}-pvc`,
          namespace: args.namespace,
          annotations: {
            'pulumi.com/skipAwait': 'true',
          },
        },
        spec: {
          accessModes: [args.accessMode],
          resources: {
            requests: {
              storage: args.size,
            },
          },
        },
      },
      {
        parent: this,
      }
    )
  }
}
