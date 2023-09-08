# Athena Cycle

Athena cycle is an SDLC (Software Development Lifecycle) tracking/management system.

## Components

1. A postgres database
2. Syncer
3. An appsmith frontend

## Installation

### Deploy the components

#### Docker-compose

1. Export the following environment variables:
   1. GitHub integration:
      1. Get a GITHUB_TOKEN
   2. Jira integration:
      1. Get your JIRA_USERNAME
      2. Get a JIRA_API_TOKEN
2. Run `docker-compose up -d`

#### Kubernetes

> This section assumes you have a kubernetes cluster and an ingress controller properly configured.

1. Have a postgres database running
2. `kubectl create ns athena-cycle`
3. Create the secrets `github-token`, `postgresql-secret`, `jira-secret`

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: github-token
    type: Opaque
      GITHUB_TOKEN: <GITHUB_TOKEN>
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: postgresql-secret
    type: Opaque
      POSTGRES_USER: <POSTGRES_USER>
      POSTGRES_PASSWORD: <POSTGRES_PASSWORD>
      DB_HOST: <DB_HOST>
      DB_NAME: <DB_NAME>
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: jira-secret
    type: Opaque
      JIRA_SITE_URL: <JIRA_SITE_URL>
      JIRA_USERNAME: <JIRA_USERNAME>
      JIRA_API_TOKEN: <JIRA_API_TOKEN>
    ```

4. Install appsmith:

    ```sh
    helm repo add appsmith https://helm.appsmith.com
    helm repo update
    helm upgrade --install appsmith appsmith/appsmith -n athena-cycle
    ```

5. Install syncer:
   1. cd to `syncer/`
   2. Populate `GITHUB_REPOSITORIES` in `deployment/syncer.yml`
   3. Run `kustomize build ./deployment | kubectl apply -f -`

6. Fork the appsmith application `https://github.com/ilaif/athena-cycle-appsmith`

7. Go to the appsmith UI and create a new application with the forked repo
