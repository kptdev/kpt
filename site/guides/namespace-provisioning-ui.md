# Namespace provisioning using UI

In this guide, we will learn how to create a kpt package from scratch using
Config as Data UI. We will also learn how to enable customization of the package
with minimal steps for package consumers and deploy the package to a Kubernetes
cluster.

## What package are we creating ?

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

Start by clicking the `Register Repository` button in the upper right corner to
register a blueprint repository. Enter the following details in the Register
Repository flow:

- In `Repository Details`, the Repository URL is the clone URL from your
  repository. Branch and directory can be left blank. Branch will default to
  `main` and directory will default to `/`.
- In `Repository Authentication`, you`ll need to use the GitHub Personal Access
  Token (unless your repository allows for unauthenticated writes). Github
  Personal Access Tokens can be created at https://github.com/settings/tokens,
  and must include the 'repo' scope.
- In `Repository Content`, select `Team Blueprints`.

Once the repository is registered, use the breadcrumbs (upper left) to navigate
back to the Repositories view.

### Registering deployment repository

Start by clicking the `Register Repository` button in the upper right corner to
register a deployment repository. Enter the following details in the Register
Repository flow:

- In `Repository Details`, the Repository URL is the clone URL from your
  repository. Branch and Directory can be left blank. Branch will default to
  `main`, and directory will default to `/`.
- In `Repository Authentication`, either create a new secret, or optionally
  select the same secret in the Authentication Secret dropdown you created for
  registering blueprint repository.
- In `Repository Content`, select Deployments.

Once the repository is created, use the breadcrumbs (upper left) to navigate
back to the Repositories view.

## Creating a Blueprint from scratch

Now that we have our repositories registered, we are ready to create our first
blueprint using the UI.

- On the Repositories Page, click the row of the team blueprint repository where
  you want to add the new blueprint to.
- Clicking the row will take you to a new screen where you can see the
  packages/blueprints in the selected repository. If this is a new repository,
  the list will be empty.
- Click the `Add Team Blueprint` button in the upper right corner to create a
  new team blueprint.
- In `Add Team Blueprint`, create a new team blueprint from scratch with the
  name `simplens`.
  ![add-blueprint](/static/images/porch-ui/blueprint/add-blueprint.png)
- After completing the above flow, you`ll be taken to your newly created
  blueprint (see screenshot below). Here you will have a chance to add, edit,
  and remove resources and functions.
  ![new-blueprint](/static/images/porch-ui/blueprint/new-blueprint.png)
- Clicking any of the resources on the table (currently the `Kptfile` and
  `ConfigMap`) will show the resource viewer dialog where you can see quick
  information for each resource and view the yaml for the resource.
- On the blueprint (see the above screenshot), click `Edit` to be able to edit
  the blueprint. After clicking `Edit`, you should see this screen where you
  have an option to add new resources.
  ![add-resources](/static/images/porch-ui/blueprint/edit-new-blueprint.png)
- Using the `Add Resource` button, add a new Namespace resource. Name the
  namespace `example`.
- Click the Kptfile resource and add a new mutator
  - Search for `namespace` and select `set-namespace` with the latest version
    available for selection.
  - Select `ConfigMap: kptfile.kpt.dev` for the function config
  - By setting both of these values, anytime the blueprint is rendered (for
    instance, on save or when a deployable instance of the blueprint is
    created), the namespace will be set to the name of the package.
- Using the `Add Resource` button, add a new Role Binding resource
  - Name the resource `app-admin`
  - In Role Reference, select `Cluster Role` and set `app-admin` as the name
  - Click Add Subject, and in the newly added subject, select `Group` and set
    the name to `example.admin@bigco.com`.
- Using the `Add Resource` button, add a new Apply Replacements resource.
  - Name the resource `update-rolebinding`
  - In Source, select `ConfigMap: kptfile.kpt.dev` as the source resource and
    set `data.name: example` as source path
  - In Target, select `RoleBinding: app-admin` as the target resource and set
    `subjects.0.name: example.admin@bigco.com` as the target path. Select
    `Partial Value` as the replace value option, with `period (.)` as the
    delimiter selecting `example` to replace.
- Click the Kptfile resource to see the `apply-replacements` mutator has been
  added automatically by the previous step
  - The UI knows to add this mutator anytime a an `ApplyReplacements` resource
    is added
  - Screenshot of the Kptifle showing the two mutators
    ![kptfile-mutators](/static/images/porch-ui/blueprint/edit-kptfile-mutators.png)
- Using the `Add Resource` button, add a new Resource Quota resource
  - Name the resource `base`
  - Set Max CPU Requests to `40` and Max Memory Requests to `40G`
- After you are done with the above, you should have the following resources
  ![new-blueprint-resources](/static/images/porch-ui/blueprint/edit-new-blueprint-resources.png)
- Clicking `Save` will save the resources, apply the mutator, and take you back
  to the blueprint screen you started on. Note that the namespace has been
  updated on the resources from the `set-namespace` mutator.
  ![saved-new-blueprint](/static/images/porch-ui/blueprint/saved-new-blueprint.png)
- Click the individual resources to see the first class editors.
- Click Propose to propose the blueprint (button will change to Approve)
- Click Approve to approve the blueprint
- Using the breadcrumbs, click back to view your blueprints repository - here
  you should see the blueprint you just created has been finalized.
  ![finalized-blueprint](/static/images/porch-ui/blueprint/finalized-blueprint.png)

So, with that, we created a blueprint from scratch and published it in a team
blueprint repository. You should be able to see the blueprint by viewing the
`git` repository directly well.

## Create a deployable instance of a blueprint

In this section, we will walk through the steps of creating a deployable
instance of a blueprint.

- Starting from the team blueprints repository, you should see the `simplens`
  blueprint created above.
  ![finalized-blueprint](/static/images/porch-ui/blueprint/finalized-blueprint.png)
- Click the `simplens` blueprint.
  ![show-blueprint](/static/images/porch-ui/deployment/show-blueprint.png)
- Click the `Clone` button in the upper right corner to take you to the clone
  flow. For Action, select
  `Create a new deployment by cloning the simplens team blueprint` and name the
  deployment `backend`.
  ![add-deployment](/static/images/porch-ui/deployment/add-deployment.png)
- Completing the flow will add the package to your deployments repository. Note
  that the namespace across all the namespace scoped resources have been updated
  to the name of the package.
  ![added-deployment](/static/images/porch-ui/deployment/backend-deployment-added.png)
- Click the `Diff` button for the `RoleBinding app-admin` resource. Here you’ll
  see the differences between this resource and the same resource in the
  upstream `simplens` team blueprint. In addition to the namespace being
  updated, the binding has also been updated from the Apply Replacements
  resource.
  ![draft-deployment-screenshot](/static/images/porch-ui/deployment/rolebinding-diff.png)
- After closing the diff dialog, using the breadcrumbs, click back the
  deployments view to see your new deployment is added in Draft status.
  ![draft-deployment-screenshot](/static/images/porch-ui/deployment/deployments-list.png)
- Click into the backend deployment and move the deployment to Proposed, then
  Published by approving the deployment. Optionally, before moving the
  deployment to Proposed, you can make changes to the deployment by
  adding/removing/updating resources.
- Once the deployment is published, click `Create Sync` to have `Config Sync`
  sync the deployment to the Kubernetes cluster.
  ![published-deployment](/static/images/porch-ui/deployment/published-deployment.png)
- After a few seconds, you`ll see the Sync status update in the upper right-hand
  corner.
  ![synced-deployment](/static/images/porch-ui/deployment/synced-deployment.png)
- If you navigate back to the `deployment` repository, you will see `sync`
  status next to each deployment instance.
  ![synced-deployment-screenshot](/static/images/porch-ui/deployment/synced-deployment-list.png)

So, this completes our end-to-end workflow of creating a blueprint from scratch
and deploying it to a Kubernetes cluster using the Config as Data UI.
