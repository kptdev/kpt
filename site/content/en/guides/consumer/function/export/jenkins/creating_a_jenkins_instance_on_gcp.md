---
title: "Creating a Jenkins Instance on GCP"
toc_hide: true
type: docs
---

1.  Create a Firewall Rule that allows TCP connections to port `8080`, which Jenkins uses.
    1.  [Go to the Firewall page](https://console.cloud.google.com/networking/firewalls).
    1.  Click `Create Firewall Rule`.
    1.  Specify a `Name` for your firewall rule, e.g. `allow-http-8080`.
    1.  Select `All instances in the network` as Targets.
    1.  Select `IP ranges` in the `Source filter`.
    1.  Enter `0.0.0.0/0` as `Source IP ranges`.
    1.  In `Protocols and ports`, use `Specified protocols and ports`, check `tcp` and input `8080`.
    1.  Click the `Create` button.
1.  [Create a new VM instance](https://cloud.google.com/compute/docs/instances/create-start-instance) of Ubuntu 18.04 LTS on GCP.
    1.  Choose `Ubuntu 18.04 LTS` as image in `Boot disk`.
    1.  Expand `Management, security, disk, networking, sole tenancy`, click the `Networking` tab, enter the name of the firewall rule you created,  e.g. `allow-http-8080`.
1.  SSH into the VM instance you just created.
1.  [Install Jenkins](https://www.jenkins.io/doc/book/installing/#linux) on the VM.
    
    1.  Install JDK first.
    
        ```shell script
        sudo apt update
        sudo apt install openjdk-8-jdk
        ```     
    
    1.  Then install Jenkins.

        ```shell script
        wget -q -O - https://pkg.jenkins.io/debian-stable/jenkins.io.key | sudo apt-key add -
        sudo sh -c 'echo deb https://pkg.jenkins.io/debian-stable binary/ > \
            /etc/apt/sources.list.d/jenkins.list'
        sudo apt-get update
        sudo apt-get install jenkins
        ```

1.  Go to `<instance ip>:8080` to set up Jenkins.
