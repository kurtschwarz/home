import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

type CertificateArgs = {
  domain: pulumi.Input<string>
  namespace: pulumi.Input<string>
}

export class Certificate extends pulumi.ComponentResource {
  public certificate: kube.apiextensions.CustomResource
  public secretName: pulumi.Output<string>

  constructor(
    private name: string,
    private args: CertificateArgs,
    private opts?: pulumi.ComponentResourceOptions
  ) {
    super('kurtschwarz:home/resources:Certificate', name, args, opts)

    this.secretName = pulumi.output(`${this.name}-certificate`)
    this.certificate = new kube.apiextensions.CustomResource(
      `${this.name}-certificate`,
      {
        apiVersion: 'cert-manager.io/v1',
        kind: 'Certificate',
        metadata: {
          namespace: this.args.namespace,
        },
        spec: {
          commonName: this.args.domain,
          secretName: this.secretName,
          dnsNames: [
            this.args.domain,
          ],
          issuerRef: {
            name: 'cert-manager-lets-encrypt-issuer',
            kind: 'ClusterIssuer',
          },
        },
      },
      {
        parent: this,
      },
    )
  }
}
