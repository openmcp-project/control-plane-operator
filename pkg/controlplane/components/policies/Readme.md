# Policies

## DeploymentRuntimeConfiguration

The DeploymentRuntimeConfigurationPolicy component applies a policy to prevent end-users from editing all fields of a crossplane DeploymentRuntimeConfig. This is done to ensure that end-users cannot run arbitrary workload by introducing new containers or changed images. Additionally this prevents users to run crossplane providers in unintended configurations, which could affect other MCP users (e.g. increasing resource limits, putting unnecessary strain on a provider by running it with undesirable configuration options).

It works in conjunction with the eso- and defaultDeploymentRuntimeConfiguration-components in the following way:

* Whenever a managed provider is being rolled out, defaultDeploymentRuntimeConfiguration will create a DeploymentRuntimeConfig for this component
* End-users are allowed to edit this DeploymentRuntimeConfig (but not create new ones)
* For every modification (patch or update), the policy will only allow the following fields of a DeploymentRuntimeConfig to be edited. All other changes, will be rejected:
  * `spec.deploymentTemplate.spec.template.spec.containers.args`
  * `spec.serviceAccountTemplate.metadata.name`
