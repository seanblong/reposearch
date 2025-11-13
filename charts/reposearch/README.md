# reposearch

Installs an instance of `reposearch`, frontend, backend (API), database, and indexer.
The templates immediately available in this directory handle building the indexer
component and will generate unique cronjobs per repo you desire to index.

The frontend and backend subcharts are based on the [generic](./generic) chart.

The PostgreSQL database (pgvector) is deployed using Bitnami's
[Helm chart](https://github.com/bitnami/charts/tree/main/bitnami/postgresql), but
leverages LinuxSuRen's [pgvector Docker image](https://github.com/LinuxSuRen/pgvector-docker).

## Parameters

### Global parameters

| Name                                  | Description                                                                                | Value  |
| ------------------------------------- | ------------------------------------------------------------------------------------------ | ------ |
| `global.imageRegistry`                | Global Docker image registry                                                               | `""`   |
| `global.imagePullSecrets`             | Global Docker registry secret names as an array                                            | `[]`   |
| `global.storageClass`                 | Global StorageClass for Persistent Volume(s)                                               | `""`   |
| `global.security.allowInsecureImages` | Allow insecure images globally (required when deviating from Bitnami chart for Postgresql) | `true` |

### Common parameters

| Name                | Description                                       | Value           |
| ------------------- | ------------------------------------------------- | --------------- |
| `kubeVersion`       | Override Kubernetes version                       | `""`            |
| `nameOverride`      | String to partially override common.names.name    | `""`            |
| `fullnameOverride`  | String to fully override common.names.fullname    | `""`            |
| `namespaceOverride` | String to fully override common.names.namespace   | `""`            |
| `commonLabels`      | Labels to add to all deployed objects             | `{}`            |
| `commonAnnotations` | Annotations to add to all deployed objects        | `{}`            |
| `clusterDomain`     | Kubernetes cluster domain name                    | `cluster.local` |
| `extraDeploy`       | Array of extra objects to deploy with the release | `[]`            |

### Reposearch Configuration parameters

| Name                   | Description                                                               | Value                                                                                    |
| ---------------------- | ------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `provider`             | Specifies the AI provider to use for embeddings and summarization         | `stub`                                                                                   |
| `providerApiKey`       | API key for the selected provider                                         | `""`                                                                                     |
| `providerProjectID`    | The project ID of your OpenAI API key or the Google Cloud project         | `""`                                                                                     |
| `providerLocation`     | The location/region for the Google Cloud services                         | `us-central1`                                                                            |
| `providerEmbedModel`   | The specific model to use for generating text embeddings                  | `""`                                                                                     |
| `providerSummaryModel` | The specific model to use for generating code summaries                   | `""`                                                                                     |
| `providerEmbedDim`     | The dimensionality of the embedding vectors                               | `0`                                                                                      |
| `database`             | The connection URL (DSN) for the PostgreSQL database                      | `postgres://postgres:postgres@{{ .Release.Name }}-postgresql:5432/repos?sslmode=disable` |
| `repoURL`              | Default Git repository URL (used if no specific jobs are defined)         | `""`                                                                                     |
| `gitRoot`              | Default local repository path (used if no specific jobs are defined)      | `.`                                                                                      |
| `gitRef`               | Default Git reference (branch, tag, or commit SHA) to check out and index | `main`                                                                                   |
| `logLevel`             | The logging level for the application                                     | `info`                                                                                   |
| `port`                 | The port for the API server to listen on                                  | `8080`                                                                                   |

### Authentication Configuration

| Name                      | Description                                 | Value                                 |
| ------------------------- | ------------------------------------------- | ------------------------------------- |
| `auth.enabled`            | Enable or disable GitHub authentication     | `false`                               |
| `auth.jwtSecret`          | JWT secret for signing tokens               | `""`                                  |
| `auth.githubClientID`     | GitHub OAuth App Client ID                  | `""`                                  |
| `auth.githubClientSecret` | GitHub OAuth App Client Secret              | `""`                                  |
| `auth.redirectURL`        | OAuth Redirect URL                          | `http://localhost:8080/auth/callback` |
| `auth.githubAllowedOrg`   | Allowed GitHub Organization for user access | `""`                                  |

### Indexer CronJob Configuration

| Name                                 | Description                                                                                                       | Value                          |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `indexer.enabled`                    | Enable indexer cronjob                                                                                            | `true`                         |
| `indexer.schedule`                   | Cron schedule for the indexer job                                                                                 | `0 2 * * *`                    |
| `indexer.timeZone`                   | Timezone for the cron schedule                                                                                    | `UTC`                          |
| `indexer.concurrencyPolicy`          | How to handle overlapping jobs                                                                                    | `Forbid`                       |
| `indexer.failedJobsHistoryLimit`     | Number of failed jobs to keep                                                                                     | `3`                            |
| `indexer.successfulJobsHistoryLimit` | Number of successful jobs to keep                                                                                 | `1`                            |
| `indexer.startingDeadlineSeconds`    | Deadline for starting the job if missed scheduled time                                                            | `60`                           |
| `indexer.activeDeadlineSeconds`      | Maximum time for job execution                                                                                    | `3600`                         |
| `indexer.backoffLimit`               | Number of retries before marking job as failed                                                                    | `2`                            |
| `indexer.ttlSecondsAfterFinished`    | Time to live for completed jobs                                                                                   | `86400`                        |
| `indexer.restartPolicy`              | Restart policy for the job pods                                                                                   | `OnFailure`                    |
| `indexer.image.registry`             | Indexer image registry                                                                                            | `ghcr.io`                      |
| `indexer.image.repository`           | Indexer image repository                                                                                          | `seanblong/reposearch/indexer` |
| `indexer.image.tag`                  | Indexer image tag                                                                                                 | `indexer-v0.0.3`               |
| `indexer.image.digest`               | Indexer image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag | `""`                           |
| `indexer.image.pullPolicy`           | Indexer image pull policy                                                                                         | `IfNotPresent`                 |
| `indexer.image.pullSecrets`          | Indexer image pull secrets                                                                                        | `[]`                           |
| `indexer.image.debug`                | Enable indexer image debug mode                                                                                   | `false`                        |

### Indexer Jobs Configuration

| Name                                                        | Description                                                                                                                                                                                                                       | Value            |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `indexer.jobs`                                              | Array of indexer job configurations                                                                                                                                                                                               | `[]`             |
| `indexer.containerPorts.http`                               | Indexer HTTP container port (not applicable for cronjobs)                                                                                                                                                                         | `8080`           |
| `indexer.podSecurityContext.enabled`                        | Enabled indexer pods' Security Context                                                                                                                                                                                            | `false`          |
| `indexer.podSecurityContext.fsGroup`                        | Set indexer pod's Security Context fsGroup                                                                                                                                                                                        | `1001`           |
| `indexer.containerSecurityContext.enabled`                  | Enabled containers' Security Context                                                                                                                                                                                              | `false`          |
| `indexer.containerSecurityContext.seLinuxOptions`           | Set SELinux options in container                                                                                                                                                                                                  | `{}`             |
| `indexer.containerSecurityContext.runAsUser`                | Set containers' Security Context runAsUser                                                                                                                                                                                        | `1001`           |
| `indexer.containerSecurityContext.runAsGroup`               | Set containers' Security Context runAsGroup                                                                                                                                                                                       | `0`              |
| `indexer.containerSecurityContext.runAsNonRoot`             | Set container's Security Context runAsNonRoot                                                                                                                                                                                     | `true`           |
| `indexer.containerSecurityContext.privileged`               | Set container's Security Context privileged                                                                                                                                                                                       | `false`          |
| `indexer.containerSecurityContext.readOnlyRootFilesystem`   | Set container's Security Context readOnlyRootFilesystem                                                                                                                                                                           | `false`          |
| `indexer.containerSecurityContext.allowPrivilegeEscalation` | Set container's Security Context allowPrivilegeEscalation                                                                                                                                                                         | `false`          |
| `indexer.containerSecurityContext.capabilities.drop`        | List of capabilities to be dropped                                                                                                                                                                                                | `["ALL"]`        |
| `indexer.containerSecurityContext.seccompProfile.type`      | Set container's Security Context seccomp profile                                                                                                                                                                                  | `RuntimeDefault` |
| `indexer.command`                                           | Override default container command (useful when using custom images)                                                                                                                                                              | `[]`             |
| `indexer.args`                                              | Override default container args (useful when using custom images)                                                                                                                                                                 | `[]`             |
| `indexer.hostAliases`                                       | indexer pods host aliases                                                                                                                                                                                                         | `[]`             |
| `indexer.podLabels`                                         | Extra labels for indexer pods                                                                                                                                                                                                     | `{}`             |
| `indexer.podAnnotations`                                    | Annotations for indexer pods                                                                                                                                                                                                      | `{}`             |
| `indexer.cronjobAnnotations`                                | Annotations for indexer cronjob                                                                                                                                                                                                   | `{}`             |
| `indexer.podAffinityPreset`                                 | Pod affinity preset. Ignored if `indexer.affinity` is set. Allowed values: `soft` or `hard`                                                                                                                                       | `""`             |
| `indexer.podAntiAffinityPreset`                             | Pod anti-affinity preset. Ignored if `indexer.affinity` is set. Allowed values: `soft` or `hard`                                                                                                                                  | `soft`           |
| `indexer.nodeAffinityPreset.type`                           | Node affinity preset type. Ignored if `indexer.affinity` is set. Allowed values: `soft` or `hard`                                                                                                                                 | `""`             |
| `indexer.nodeAffinityPreset.key`                            | Node label key to match. Ignored if `indexer.affinity` is set                                                                                                                                                                     | `""`             |
| `indexer.nodeAffinityPreset.values`                         | Node label values to match. Ignored if `indexer.affinity` is set                                                                                                                                                                  | `[]`             |
| `indexer.affinity`                                          | Affinity for indexer pods assignment                                                                                                                                                                                              | `{}`             |
| `indexer.nodeSelector`                                      | Node labels for indexer pods assignment                                                                                                                                                                                           | `{}`             |
| `indexer.tolerations`                                       | Tolerations for indexer pods assignment                                                                                                                                                                                           | `[]`             |
| `indexer.updateStrategy.type`                               | indexer statefulset strategy type                                                                                                                                                                                                 | `RollingUpdate`  |
| `indexer.priorityClassName`                                 | indexer pods' priorityClassName                                                                                                                                                                                                   | `""`             |
| `indexer.topologySpreadConstraints`                         | Topology Spread Constraints for pod assignment spread across your cluster among failure-domains. Evaluated as a template                                                                                                          | `[]`             |
| `indexer.schedulerName`                                     | Name of the k8s scheduler (other than default) for indexer pods                                                                                                                                                                   | `""`             |
| `indexer.terminationGracePeriodSeconds`                     | Seconds Redmine pod needs to terminate gracefully                                                                                                                                                                                 | `""`             |
| `indexer.lifecycleHooks`                                    | for the indexer container(s) to automate configuration before or after startup                                                                                                                                                    | `{}`             |
| `indexer.extraEnvVars`                                      | Array with extra environment variables to add to indexer nodes                                                                                                                                                                    | `[]`             |
| `indexer.extraEnvVarsCM`                                    | Name of existing ConfigMap containing extra env vars for indexer nodes                                                                                                                                                            | `""`             |
| `indexer.extraEnvVarsSecret`                                | Name of existing Secret containing extra env vars for indexer nodes                                                                                                                                                               | `""`             |
| `indexer.extraVolumes`                                      | Optionally specify extra list of additional volumes for the indexer pod(s)                                                                                                                                                        | `[]`             |
| `indexer.extraVolumeMounts`                                 | Optionally specify extra list of additional volumeMounts for the indexer container(s)                                                                                                                                             | `[]`             |
| `indexer.sidecars`                                          | Add additional sidecar containers to the indexer pod(s)                                                                                                                                                                           | `[]`             |
| `indexer.initContainers`                                    | Add additional init containers to the indexer pod(s)                                                                                                                                                                              | `[]`             |
| `indexer.resourcesPreset`                                   | Set container resources according to one common preset (allowed values: none, nano, micro, small, medium, large, xlarge, 2xlarge). This is ignored if indexer.resources is set (indexer.resources is recommended for production). | `medium`         |
| `indexer.resources`                                         | Set container requests and limits for different resources like CPU or memory (essential for production workloads)                                                                                                                 | `{}`             |
| `indexer.autoscaling.hpa.enabled`                           | Enable Horizontal Pod Autoscaler (HPA) for indexer (not applicable for cronjobs)                                                                                                                                                  | `false`          |
| `indexer.automountServiceAccountToken`                      | Mount Service Account token in pod                                                                                                                                                                                                | `false`          |

### Persistence Parameters

| Name                        | Description                                                                                             | Value   |
| --------------------------- | ------------------------------------------------------------------------------------------------------- | ------- |
| `persistence.enabled`       | Enable persistence using Persistent Volume Claims                                                       | `true`  |
| `persistence.mountPath`     | Path to mount the volume at.                                                                            | `/data` |
| `persistence.subPath`       | The subdirectory of the volume to mount to, useful in dev environments and one PV for multiple services | `""`    |
| `persistence.storageClass`  | Storage class of backing PVC                                                                            | `""`    |
| `persistence.annotations`   | Additional custom annotations for the PVC                                                               | `{}`    |
| `persistence.labels`        | Additional custom labels for the PVC                                                                    | `{}`    |
| `persistence.accessModes`   | Persistent Volume access modes                                                                          | `[]`    |
| `persistence.size`          | Size of data volume                                                                                     | `8Gi`   |
| `persistence.existingClaim` | The name of an existing PVC to use for persistence                                                      | `""`    |
| `persistence.selector`      | Selector to match an existing Persistent Volume for the indexer data PVC                                | `{}`    |
| `persistence.dataSource`    | Custom PVC data source                                                                                  | `{}`    |

### Service Account Parameters

| Name                                          | Description                                                                                | Value   |
| --------------------------------------------- | ------------------------------------------------------------------------------------------ | ------- |
| `serviceAccount.create`                       | Specifies whether a ServiceAccount should be created                                       | `true`  |
| `serviceAccount.name`                         | The name of the ServiceAccount to use.                                                     | `""`    |
| `serviceAccount.automountServiceAccountToken` | Automount service account token for the server service account                             | `false` |
| `serviceAccount.annotations`                  | Annotations for service account. Evaluated as a template. Only used if `create` is `true`. | `{}`    |

### Other Parameters

| Name                     | Description                                                                             | Value          |
| ------------------------ | --------------------------------------------------------------------------------------- | -------------- |
| `diagnosticMode.enabled` | Enable diagnostic mode (all probes will be disabled and the command will be overridden) | `false`        |
| `diagnosticMode.command` | Command to override all containers in the deployment                                    | `["sleep"]`    |
| `diagnosticMode.args`    | Args to override all containers in the deployment                                       | `["infinity"]` |

### API Subchart Configuration

| Name                      | Description                                                                                                                                                                                                                       | Value                      |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `api.image.registry`      | API image registry                                                                                                                                                                                                                | `ghcr.io`                  |
| `api.image.repository`    | API image repository                                                                                                                                                                                                              | `seanblong/reposearch/api` |
| `api.image.tag`           | API image tag                                                                                                                                                                                                                     | `api-v0.0.3`               |
| `api.image.pullPolicy`    | API image pull policy                                                                                                                                                                                                             | `IfNotPresent`             |
| `api.replicaCount`        | Number of API replicas                                                                                                                                                                                                            | `1`                        |
| `api.containerPorts.http` | API HTTP container port                                                                                                                                                                                                           | `8080`                     |
| `api.service.type`        | API service type                                                                                                                                                                                                                  | `ClusterIP`                |
| `api.service.ports.http`  | API service HTTP port                                                                                                                                                                                                             | `8080`                     |
| `api.extraEnvVarsCM`      | Use the shared reposearch ConfigMap for API configuration                                                                                                                                                                         | `{{ .Release.Name }}`      |
| `api.extraEnvVarsSecret`  | Name of existing Secret containing extra env vars for API containers                                                                                                                                                              | `github-token`             |
| `api.resourcesPreset`     | Set container resources according to one common preset (allowed values: none, nano, micro, small, medium, large, xlarge, 2xlarge). This is ignored if indexer.resources is set (indexer.resources is recommended for production). | `micro`                    |
| `api.resources`           | Set API container resources                                                                                                                                                                                                       | `{}`                       |
| `api.ingress.enabled`     | Enable ingress for API                                                                                                                                                                                                            | `false`                    |
| `api.ingress.hostname`    | API ingress hostname                                                                                                                                                                                                              | `reposearch-api.local`     |
| `api.ingress.path`        | API ingress path                                                                                                                                                                                                                  | `/api`                     |

### Frontend Subchart Configuration

| Name                                        | Description                                                                                                                                                                                                                       | Value                           |
| ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| `frontend.image.registry`                   | Frontend image registry                                                                                                                                                                                                           | `ghcr.io`                       |
| `frontend.image.repository`                 | Frontend image repository                                                                                                                                                                                                         | `seanblong/reposearch/frontend` |
| `frontend.image.tag`                        | Frontend image tag                                                                                                                                                                                                                | `frontend-v0.0.3`               |
| `frontend.image.pullPolicy`                 | Frontend image pull policy                                                                                                                                                                                                        | `IfNotPresent`                  |
| `frontend.replicaCount`                     | Number of frontend replicas                                                                                                                                                                                                       | `1`                             |
| `frontend.containerPorts.http`              | Frontend HTTP container port                                                                                                                                                                                                      | `80`                            |
| `frontend.service.type`                     | Frontend service type                                                                                                                                                                                                             | `ClusterIP`                     |
| `frontend.service.ports.http`               | Frontend service HTTP port                                                                                                                                                                                                        | `80`                            |
| `frontend.resourcesPreset`                  | Set container resources according to one common preset (allowed values: none, nano, micro, small, medium, large, xlarge, 2xlarge). This is ignored if indexer.resources is set (indexer.resources is recommended for production). | `micro`                         |
| `frontend.resources`                        | Set frontend container resources                                                                                                                                                                                                  | `{}`                            |
| `frontend.containerSecurityContext.enabled` | Enable default container security context for frontend containers                                                                                                                                                                 | `false`                         |
| `frontend.extraEnvVars`                     | Extra environment variables for frontend containers                                                                                                                                                                               | `[]`                            |
| `frontend.ingress.enabled`                  | Enable ingress for frontend                                                                                                                                                                                                       | `false`                         |
| `frontend.ingress.hostname`                 | Frontend ingress hostname                                                                                                                                                                                                         | `reposearch.local`              |
| `frontend.ingress.path`                     | Frontend ingress path                                                                                                                                                                                                             | `/`                             |

### PostgreSQL Configuration (subchart)

| Name                                                  | Description                                                      | Value                 |
| ----------------------------------------------------- | ---------------------------------------------------------------- | --------------------- |
| `postgresql.enabled`                                  | Enable PostgreSQL subchart                                       | `true`                |
| `postgresql.image.registry`                           | PostgreSQL image registry                                        | `ghcr.io`             |
| `postgresql.image.repository`                         | PostgreSQL image repository                                      | `linuxsuren/pgvector` |
| `postgresql.image.tag`                                | PostgreSQL image tag                                             | `v0.0.1`              |
| `postgresql.primary.podSecurityContext.enabled`       | Enable default pod security context for PostgreSQL primary       | `false`               |
| `postgresql.primary.containerSecurityContext.enabled` | Enable default container security context for PostgreSQL primary | `false`               |
| `postgresql.auth.postgresPassword`                    | PostgreSQL admin password                                        | `postgres`            |
| `postgresql.auth.username`                            | PostgreSQL username                                              | `postgres`            |
| `postgresql.auth.password`                            | PostgreSQL user password                                         | `postgres`            |
| `postgresql.auth.database`                            | PostgreSQL database name                                         | `repos`               |
