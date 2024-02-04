import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

import { TwingateConnector } from './resources/connector'

interface Output {
  namespace: pulumi.Output<string>
}

export = async function (): Promise<Output> {
  const config = new pulumi.Config('twingate')
  const namespace = new kube.core.v1.Namespace('twingate-namespace', {
    metadata: {
      name: 'twingate',
    },
  })

  for (const connector of config.requireObject<{ name: string, accessToken: string, refreshToken: string }[]>('connectors')) {
    new TwingateConnector(
      `twingate-connector-${connector.name}`,
      {
        image: config.require('image'),
        namespace: namespace.metadata.name,
        tenantUrl: config.requireSecret('tenantUrl'),
        accessToken: connector.accessToken,
        refreshToken: connector.refreshToken,
      },
      {
        parent: namespace,
      },
    )
  }

  return {
    namespace: namespace.metadata.name,
  }
}
