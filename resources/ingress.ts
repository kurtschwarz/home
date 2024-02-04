import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

import { Certificate } from './certificate'

export enum IngressType {
  HTTP = 'http',
}

export enum AccessType {
  Public = 'public',
  Private = 'private',
}

type IngressArgs = {
  type: IngressType
  access: AccessType
  domain: pulumi.Input<string>
  namespace: pulumi.Input<string>
  serviceName: pulumi.Input<string>
  servicePort: pulumi.Input<number>
}

export class Ingress extends pulumi.ComponentResource {
  private types: Map<IngressType, () => void> = new Map([
    [IngressType.HTTP, this._httpIngressResources],
  ])

  private certificate?: Certificate

  constructor(
    private name: string,
    private args: IngressArgs,
    private opts?: pulumi.ComponentResourceOptions
  ) {
    super('kurtschwarz:home/resources:Ingress', name, args, opts)

    this.types.get(args.type)?.call(this)
  }

  private _httpIngressResources(): void {
    this.certificate = new Certificate(
      this.name,
      {
        domain: this.args.domain,
        namespace: this.args.namespace,
      },
    )

    new kube.apiextensions.CustomResource(
      `${this.name}-http-ingress-route`,
      {
        apiVersion: 'traefik.containo.us/v1alpha1',
        kind: 'IngressRoute',
        metadata: {
          name: `${this.name}-http`,
          namespace: this.args.namespace,
        },
        spec: {
          entryPoints: [
            'web',
            'web-secure',
          ],
          routes: [
            {
              match: `Host(\`${this.args.domain}\`)`,
              kind: 'Rule',
              services: [
                {
                  name: this.args.serviceName,
                  port: this.args.servicePort,
                },
              ],
            },
          ],
          tls: {
            secretName: this.certificate.secretName
          },
        },
      },
      {
        parent: this,
        dependsOn: [
          this.certificate,
        ],
      },
    )
  }
}
