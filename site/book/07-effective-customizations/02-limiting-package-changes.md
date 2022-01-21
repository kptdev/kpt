## Scenario:

I’d like to limit what my package consumers can do with my package and it feels safer to just provide a string replacement in one place so they know not to alter the configuration outside of the few places that I designated as OK places to change.

## Problems:

1. The limitation by parameters does not guarantee that consumers are in fact going to limit their changes to the parameters.  A popular pattern is using kustomize to change Helm packages beyond what the package author has allowed by parameters.
1. String replacements rarely describe the intent of the package author.
When additional resources are added I need additional places where parameters need to be applied.

## Solutions:

1. General ways to describe policy already exist.  kpt has a [gatekeeper] function that allows the author to describe intended limitations for a class of resources or the entire package giving the consumer the freedom to customize and get an error or a warning when the policy is violated. 

[gatekeeper]: https://catalog.kpt.dev/gatekeeper/v0.2/
