import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

interface Output {
  namespace: pulumi.Output<string>
}

export = async function (): Promise<Output> {
  const config = new pulumi.Config('longhorn')

  const manifest = new kube.yaml.ConfigFile(
    'longhorn-manifest',
    {
      file: `https://raw.githubusercontent.com/longhorn/longhorn/v${config.require(
        'version'
      )}/deploy/longhorn.yaml`,
      transformations: [],
    },
    {}
  )

  const namespace = manifest.getResource('v1/Namespace', 'longhorn-system')

  new kube.core.v1.Secret(
    'longhorn-s3-compatible-backup-target-secret',
    {
      metadata: {
        name: 'longhorn-s3-compatible-backup-target-secret',
        namespace: namespace.metadata.name,
      },
      type: 'Opaque',
      stringData: config.requireSecretObject('s3CompatibleBackupSecrets'),
    },
    {
      parent: manifest,
      dependsOn: [namespace],
    }
  )

  return {
    namespace: namespace.metadata.name,
  }
}
