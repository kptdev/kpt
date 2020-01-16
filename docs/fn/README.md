## fn

Run local containers against Resource configuration

### Synopsis

Functions are executables packaged in container images which accept a collection of
Resource configuration as input, and emit a collection of Resource configuration as output.

They may be used to:

- Generate configuration from Templates, DSLs, CRD-style abstractions, key-value pairs, etc. -- e.g.
  expand Helm charts, JSonnet, etc.
- Inject fields or otherwise modifying configuration -- e.g. add init-containers, side-cars, etc
- Rollout configuration changes across an organization -- e.g. similar to
  https://github.com/reactjs/react-codemod
- Validate configuration -- e.g. ensure Organizational policies are enforced

Functions may be run either imperatively with `kpt run DIR/ --image` or declaratively with
`kpt run DIR/` and specifying them in files.

Functions specified in files must contain an annotation to mark them as function declarations:

      annotations:
        config.kubernetes.io/function: |
          container:
            image: gcr.io/example.com/image:version
        config.kubernetes.io/local-config: "true"

Functions may be run at different times depending on the function and the organizational needs:

- as part of the build and development process
- as pre-commit checks
- as PR checks
- as pre-release checks
- as pre-rollout checks
