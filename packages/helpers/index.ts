import * as pulumi from '@pulumi/pulumi'
import * as kube from '@pulumi/kubernetes'

export * from './constants'
export * from './longhorn'

export const getNamespaceFromProject = (
  project: string,
  env: string
): pulumi.Output<kube.core.v1.Namespace> => {
  const stack = new pulumi.StackReference(`kurtschwarz/${project}/${env}`)

  return pulumi.output(
    kube.core.v1.Namespace.get(
      `${project}-namespace`,
      stack.getOutput('namespace')
    )
  )
}
