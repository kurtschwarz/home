import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

import { Certificate } from './certificate'

export enum IngressType {
  HTTP = 'http',
  TCP = 'tcp',
  UDP = 'udp',
}

export enum AccessType {
  Public = 'public',
  Private = 'private',
}

type IngressArgs = {
  type: IngressType
  access?: AccessType
  domain?: pulumi.Input<string>
  namespace: pulumi.Input<string>
  serviceName: pulumi.Input<string>
  servicePort: pulumi.Input<number>
  entryPoints?: pulumi.Input<string[]>
}

export class Ingress extends pulumi.ComponentResource {
  private types: Map<IngressType, () => void> = new Map([
    [IngressType.HTTP, this._httpIngressResources],
    [IngressType.TCP, this._tcpIngressResources],
    [IngressType.UDP, this._udpIngressResources],
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
    if (!this.args.domain) {
      return
    }

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

  private _tcpIngressResources(): void {
    new kube.apiextensions.CustomResource(
      `${this.name}-tcp-ingress-route`,
      {
        apiVersion: 'traefik.containo.us/v1alpha1',
        kind: 'IngressRouteTCP',
        metadata: {
          name: `${this.name}-tcp`,
          namespace: this.args.namespace,
        },
        spec: {
          entryPoints: this.args.entryPoints,
          routes: [
            {
              match: `HostSNI(\`*\`)`,
              kind: 'Rule',
              services: [
                {
                  name: this.args.serviceName,
                  port: this.args.servicePort,
                },
              ],
            },
          ],
        },
      },
      {
        parent: this
      },
    )
  }

  private _udpIngressResources(): void {
    new kube.apiextensions.CustomResource(
      `${this.name}-udp-ingress-route`,
      {
        apiVersion: 'traefik.containo.us/v1alpha1',
        kind: 'IngressRouteUDP',
        metadata: {
          name: `${this.name}-udp`,
          namespace: this.args.namespace,
        },
        spec: {
          entryPoints: this.args.entryPoints,
          routes: [
            {
              match: `HostSNI(\`*\`)`,
              kind: 'Rule',
              services: [
                {
                  name: this.args.serviceName,
                  port: this.args.servicePort,
                },
              ],
            },
          ],
        },
      },
      {
        parent: this
      },
    )
  }
}
