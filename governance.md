# Governance

The Kpt governance model is modeled after [Kubernetes](https://github.com/kubernetes/community/blob/master/governance.md). In particular, we leverage the same hierarchy of steering committee, SIGs, subprojects, working groups that Kubernetes does to facilitiate engagement with the greater ecosystem. We plan to refine this over time as the number of contributors grow. As an example, we don't plan to record all sig meetings at the start but may over time as it becomes valuable for interested parties who miss the meetings. If something is not defined here, it is expected that we will inherit the Kubernetes equivalent.

# Committees and Groups

## Kpt Steering Committee
The Kpt Steering Committee is the governing body for the Kpt project
Please reach out to us on slack or attend one of the SIG meetings to bring up anything you'd like to discuss. We actively partcipate in the SIGs as well as Office Hours
 | Name | Profile |
 | ---- | ------- |
 | Chris Sosa | **[@selfmanagingresource](https://github.com/selfmanagingresource)** |
 | Brian Grant | **[@bgrant0607](https://github.com/bgrant0607)** |
 | Justin Santa Barbara | **[@justinsb](https://github.com/justinsb)** |
 | John Belamaric | **[@johnbelamaric](https://github.com/johnbelamaric)** |
    

## SIG - Config As Data
Covers leveraging [Config as Data](https://cloud.google.com/blog/products/containers-kubernetes/understanding-configuration-as-data-in-kubernetes) to customize and automate configuration at scale. This is the largest SIG in Kpt that spans most of its surface area including incubation for apply config as data to new use cases.
### Chairs
 * Brian Grant ( **[@bgrant0607](https://github.com/bgrant0607)**)
 * Chris Sosa (**[@selfmanagingresource](https://github.com/selfmanagingresource)**)

### Sub-Projects

#### Kpt-Core
Everything not covered by another sub-project including porch, fn render, etc

##### Tech Leads

 * Sunil Arora ( **[@droot](https://github.com/droot)**)
 * Morten Torkildsen (**[@mortent](https://github.com/mortent)**)

#### Kpt SDK
Owns the SDK that enables our Config as Data experience

##### Tech Leads
  * Yuwen Ma ( **[@yuwenma](https://github.com/yuwenma)**)

### Working-Groups

#### App Wow
Focused on making it easy to leverage Config as Data with application workloads

##### Lead
 * Chris Sosa (**[@selfmanagingresource](https://github.com/selfmanagingresource)**)


## SIG - Config Sync
Focused on making it easy to automate deployments through GitOps. Owns the APIs and tools that are part of the [Config Sync repository](https://kpt.dev/gitops/configsync/)

### Chairs
 * Mike Borozdin ( **[@mikebz](https://github.com/mikebz)**)
 * Janet Kuo (**[@janetkuo](https://github.com/janetkuo)**)




