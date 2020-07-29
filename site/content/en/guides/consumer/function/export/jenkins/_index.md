---
title: 'Exporting a Jenkins Pipeline'
linkTitle: 'Jenkins'
weight: 3
type: docs
no_list: true
description: >
  Export a Jenkinsfile that runs kpt functions on Jenkins
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [Jenkins]. To make the generated pipeline work, extra steps that you may need is covered in the tutorial. This tutorial takes about 20~25 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*New to Jenkins? Here is a [tutorial]*.

Before diving into the following tutorial, you need to create a public repo on GitHub if you don't have one yet, e.g. `function-export-example`.

On your local machine, create an empty directory:

```shell script
mkdir function-export-example
cd function-export-example
```

{{% pageinfo color="warning" %}}
All commands must be run at the root of this directory.
{{% /pageinfo %}}

Use `kpt pkg get` to fetch source files of this tutorial:

```shell script
# Init git
git init
git remote add origin https://github.com/<USER>/<REPO>.git
# Fetch source files
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/function-export-blueprint example-package
```

Then you will get an `example-package` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions from [Kpt Functions Catalog] declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

## Creating a Jenkins instance

If you do not have a Jenkins instance yet, you can refer to this [page] to create one on GCP step by step, or launch a prebuilt `Jenkins VM` instance on the [Marketplace].

## Installing Docker on the Jenkins Agents

The exported pipeline leverages docker to run the kpt container, so you also need to [install docker] on the Jenkins agents.

1. Install docker using the convenience script.

    ```shell script
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    ```

1. Add the `jenkins` user to the `docker` group so that docker commands can be run in Jenkins pipelines.

    ```shell script
    sudo usermod -aG docker jenkins
    ```

1. Reboot the VM to let it take effect.

## Create a project on Jenkins

1. Go to `<instance ip>: 8080`, click `New Item` on the left sidebar to create a new project.
1. Enter `function-export-example` as name, select `Pipeline`, and click `OK`.
1. In the newly created project, click `Configure` to set up.
1. In the `Pipeline` section, select `Pipeline script from SCM` as `Definition`, `Git` as `SCM`, and your repo url as `Repository URL`.
1. Click `Save` at the bottom.

## Exporting a pipeline

```shell script
kpt fn export example-package --workflow jenkins --output Jenkinsfile
```

Running this command on your local machine will get a Jenkinsfile like this:

```
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/example-package
                '''
            }
        }
    }
}
```

## Integrating with your existing pipeline

Now you can manually copy and paste the `Run kpt functions` stage in the `Jenkinsfile` into your existing pipeline. This stage can be run with any [agent] as long as docker is installed on that agent, and your `jenkins` user is added to the `docker` group to access `docker.sock` file on the host. Basically, you do not have to do anything if you follow the instructions from the beginning as it is covered.

If you do not have one, you can simply copy the exported `Jenkinsfile` into your project root. It is fully functional.

If you want to see the diff after running kpt functions, append a `git diff` step . Your final `Jenkinsfile` looks like this:

```
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/example-package
                '''

                sh '''
                    git diff
                '''
            }
        }
    }
}
```

## Viewing the result on Jenkins

```shell script
git add .
git commit -am 'Init pipeline'
git push --set-upstream origin master
```

Once the changes are committed and pushed, you can go to your Jenkins server at `<instance ip>:8080` and click `Build Now` on the left sidebar in your project page. Then you will see a latest job like this:

{{< png src="images/fn-export/jenkins-result" >}}

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Once local changes are pushed, build again on Jenkins, then you can see how it fails.

[Jenkins]: https://www.jenkins.io/
[tutorial]: https://www.jenkins.io/doc/tutorials/
[Kpt Functions Catalog]: ../../catalog
[page]: ./creating_a_jenkins_instance_on_gcp
[Marketplace]: https://console.cloud.google.com/marketplace/browse?q=jenkins&filter=solution-type:vm&filter=price:free
[agent]: https://www.jenkins.io/doc/book/glossary/#agent
[install docker]: https://docs.docker.com/engine/install/ubuntu/#install-using-the-convenience-script
