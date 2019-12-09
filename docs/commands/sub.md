## sub

Perform package value substitutions

### Synopsis

Perform package value substitutions.

`sub` looks for possible value substitutions in a package by reading the Kptfile.
To print the available substitutions for a package, run `sub` on the package directory
and they will be listed as sub commands.

  PKG_DIR

    A directory containing a Kptfile with substitutions specified.

  SUBSTITUTION_NAME

    The name of the substitution to perform.  Available substitutions names will
    be listed when running `sub` against the PKG_DIR with no other arguments.

  NEW_VALUE

    The new value to substitute for the marker.

The following is an example Kptfile containing a substitution, substituting an
int specified as a commandline arg for the string `$[PORT]` in the provided field paths.

    # my-package/Kptfile
    apiVersion: kpt.dev/v1alpha1
    kind: KptFile
    substitutions:
    - name: 'port'
      type: int
      marker: '$[PORT]'
      paths: # paths to fields to substitute
      - path: ['spec', 'ports', '[name=http]', 'port']
      - path: ['spec', 'ports', '[name=http]', 'targetPort']
      long: 'long description of this substitution command'
      example: 'example of this substitution command'
      description: 'short description of this substitution command'

The preceding would enable the command: `kpt sub my-package/ port PORT_NUM`

### Examples

    # print the substitution commands for a package
    kpt sub my-package/
    ...
    Available Commands:
      port        $[PORT] (int) port and targetPort to substitute
    ...

    # print help for the port substitution
    kpt sub my-package/ port

    # perform the port substitution in my-package
    kpt sub my-package/ port 8080
