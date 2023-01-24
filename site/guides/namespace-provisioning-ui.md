# Namespace provisioning using UI

This guide will teach us how to create a kpt package from scratch using Config
as Data UI. We will also learn how to enable customization of the package with
minimal steps for package consumers and deploy the package to a Kubernetes
cluster.

## What package are we creating?

Onboarding a new application or a micro-service is a very common task for a
platform team. It involves provisioning a dedicated namespace (and other
associated resources) where all resources that belong to the application reside.
In this guide, we will create a package that will be used for provisioning a
namespace.

## Prerequisites

Before we begin, ensure that you can access the Config as Data UI. The UI is
provided as a [Backstage](https://backstage.io) plugin and can be accessed at
`[BACKSTAGE_BASE_URL]/config-as-data`.

_If you don't have a UI installed, follow the steps in the
[UI installation guide](guides/porch-ui-installation.md) to install the UI._

You will also need two new git repositories. The first repository will be where
the reusable kpt packages will live. The second repository is where the
instances of packages that will be deployed to a Kubernetes cluster will live.
Users will create deployable packages from the packages in the blueprint repo.

## Repository Registration

### Registering blueprint repository

Start by clicking the **Register Repository** link on the Team Blueprints card
to register a blueprint repository. Enter the following details in the Register
Repository flow:

- In the Repository Details section, the Repository URL is the clone URL from
  your repository. Branch and directory can be left blank. Branch will default
  to "main" and directory will default to "/".
- In the Repository Authentication section, you will need to use the GitHub
  Personal Access Token (unless your repository allows for unauthenticated
  writes). GitHub Personal Access Tokens can be created at
  <https://github.com/settings/tokens>, and must include the 'repo' scope.
- In the Repository Content section, select "Team Blueprints".

Once the repository is registered, use the breadcrumbs (upper left) to navigate
to the **Package Management** screen.

### Registering deployment repository

Start by clicking the **Register Repository** link on the Deployments card to
register a deployment repository. Enter the following details in the Register
Repository flow:

- In the Repository Details section, the Repository URL is the clone URL from
  your repository. Branch and Directory can be left blank. Branch will default
  to "main", and directory will default to "/".
- In the Repository Authentication section, either create a new secret, or
  optionally select the same secret in the Authentication Secret dropdown you
  created for registering the blueprint repository.
- In the Repository Content section, select "Deployments".

Once the repository is created, use the breadcrumbs (upper left) to navigate to
the **Package Management** screen.

## Creating a Blueprint from scratch

Now that we have our repositories registered, we are ready to create the simple
namespace team blueprint using the UI.

- From the **Package Management** screen, click the **Team Blueprints →** link.
- Clicking the link will take you to a Team Blueprints screen where you can view
  and add team blueprints.
- Click the **ADD TEAM BLUEPRINT** button in the upper right corner to create a
  new team blueprint.
- On the Add Team Blueprint screen:
  - Choose "Create a new team blueprint from scratch".
  - Click Next to proceed to the Metadata section.
  - Set the team blueprint name to "simplens".
  - Click Next to proceed to the Namespace section.
  - Check the "Add namespace resource to the team blueprint" checkbox. Checking
    this checkbox will automatically add a Namespace resource to the team
    blueprint.
  - Set namespace option to "Set the namespace to the name of the deployment
    instance". This option will set the name of the namespace equal to the name
    of the deployment package when a deployment package is created/cloned from
    this team blueprint by adding the set-namespace mutator to the Kptfile
    resource. Since we are creating a team blueprint, the namespace will be set
    to "example".
  - Click Next to proceed to the Validate Resources section.
  - Click Next to proceed to the Confirm section.
  - Click the **CREATE TEAM BLUEPRINT** button.
- The next screen is the package editor screen for the new _Draft_ team
  blueprint package you have just created. Here we will want to add a few
  resources to complete the team blueprint.
  - Click the **EDIT** button to allow the team blueprint to be updated.
  - Click the **ADD BUTTON** button and add a new Role Binding resource.
    - In the Resource Metadata section, name the resource "app-admin".
    - In Role Reference section, set the kind to "Cluster Role" and name to
      "app-admin" .
    - Click the **ADD SUBJECT** button, and in the newly added subject section,
      set the kind to "Group" and name to "example.admin@bigco.com".
    - Click the **SAVE** button to close the add resource dialog.
  - Click the **ADD BUTTON** button and add a new Apply Replacements resource.
    - In the Resource Metadata section, name the resource "update-rolebinding"
    - In the Source section, set the source resource as "ConfigMap:
      kptfile.kpt.dev" and source path to "data.name: example"
    - In the Target section, set the target resource to "RoleBinding:
      app-admin", the target path to "subjects.0.name: example.admin@bigco.com".
      Set the replacement value option to "Partial value", delimiter to "period
      (.)", and replace to "example".
    - Click the **SAVE** button to close the add resource dialog.
  - Click the **ADD BUTTON** button and add a new Resource Quota resource.
    - In the Resource Metadata section, name the resource "base".
    - In the Compute Resources setion, set Max CPU Requests to "40" and Max
      Memory Requests to "40G".
    - Click the **SAVE** button to close the add resource dialog.
  - Click the **SAVE** button on the upper right corner of the screen.
- This will bring you back to the team blueprint screen. The team blueprint is
  still in _Draft_ status so we will need to publish it to make it available for
  deployment.
  - Click the **PROPOSE** button to propose the simplens team blueprint package
    for review.
  - It will change to **APPROVE** momentarily. Click that button to publish the
    simplens team blueprint.
- Using the breadcrumbs, click back to view your blueprints repository - here
  you should see the blueprint you just created has been published.

This completes the simplens team blueprint. The next section shows how
deployable instances can be created using this team blueprint.

## Create a deployable instance of the team blueprint

In this section, we will walk through creating a deployable instance of the
simplens team blueprint.

- From the **Package Management** screen, click the **Team Blueprints →** link.
- Here you should see the "simplens" team blueprint created in the section
  above.
- Click the "simplens" team blueprint row to view the blueprint.
- Click the **CLONE** button.
- In the **Clone simplens** screen:
  - Choose "Create a new deployment by cloning the simplens team blueprint"
    option.
  - Click Next to proceed to the Metadata section.
  - Update the deployment name to "backend".
  - Click Next to proceed to the Namespace section.
  - Click Next to proceed to the Validate Resources section.
  - Click Next to proceed to the Confirm section.
  - Click the **CREATE DEPLOYMENT** button.
- The next screen is the package editor screen for the new _Draft_ deployment
  package you have just created. Note that the namespace across all the
  namespace scoped resources has been updated to the name of the package.
  - Click the **DIFF** button for any resource to view the differences between
    the resource and the same resource in the upstream simplens team blueprint.
    For many of the deployable resources, only the namespace will be updated.
    For the app-admin RoleBiding resource, the group binding has also been
    updated by the Apply Replacements resource.
- We need to publish the deployment package to make it available for deployment.
  - Before moving the deployment to _Proposed_, you can make changes by editing
    the deployment and adding, removing, and updating resources.
  - Click the **PROPOSE** button to propose the backend deployment package for
    review.
  - It will change to **APPROVE** momentarily. Click that button to publish the
    backend deployment package.
- Once the deployment package is published, click the **CREATE SYNC** button to
  have _Config Sync_ sync the deployment to the Kubernetes cluster.
  - After a few seconds, you will see the Sync status update in the upper
    right-hand corner.
- Navigating back to the deployment repository the _SYNC STATUS_ column shows
  the status of each deployment package.

This completes our end-to-end workflow of creating a blueprint from scratch and
deploying a variant of the blueprint to a Kubernetes cluster using the Config as
Data UI.
