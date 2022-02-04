## Scenario:

I’d like to limit what my package consumers can do with my package and it feels 
safer to just provide a string replacement in one place so they know not to 
alter the configuration outside of the few places that I designated as OK 
places to change.

Example deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deploy
  name: nginx-deploy
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: backend
          image: nginx:1.16.1 # kpt-set: nginx:${tag}
```

kpt configuration that uses a setter:
```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: dont-change-much
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2.0
      configMap:
        tag: 1.21
```

## Problems:

1. The limitation by parameters does not guarantee that consumers are in fact 
going to limit their changes to the parameters.  A popular pattern is using 
kustomize to change output of other tools no matter what parameters had.  In 
this particular case I am able to fork or patch this package and add:

```yaml
securityContext:
    runAsNonRoot: false
```

2. String replacements rarely describe the intent of the package author.
When additional resources are added I need additional places where parameters 
need to be applied.  I can easily add other containers to this deployment and
the package author's rules are not clear and not easily validated.

## Solutions:

1. General ways to describe policy already exist.  kpt has a [gatekeeper] 
function that allows the author to describe intended limitations for a class 
of resources or the entire package giving the consumer the freedom to customize 
and get an error or a warning when the policy is violated. 

In the sample provided by the function we see how to provide a policy that will
clearly describe the intent using rego:

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata: # kpt-merge: /disallowroot
  name: disallowroot
spec:
  crd:
    spec:
      names:
        kind: DisallowRoot
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |-
        package disallowroot
        violation[{"msg": msg}] {
          not input.review.object.spec.template.spec.securityContext.runAsNonRoot
          msg := "Containers must not run as root"
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: DisallowRoot
metadata: # kpt-merge: /disallowroot
  name: disallowroot
spec:
  match:
    kinds:
      - apiGroups:
          - 'apps'
        kinds:
          - Deployment
```

The Kptfile can enforce that resources comply with this policy every time
`kpt fn render` is used:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: gatekeeper-disallow-root-user
pipeline:
  validators:
    - image: gcr.io/kpt-fn/gatekeeper:v0.2
```

[gatekeeper]: https://catalog.kpt.dev/gatekeeper/v0.2/
