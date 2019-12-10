## sub

Perform package value substitutions

### Synopsis

Perform package value substitutions, replacing markers with new values.

`sub` looks for possible value substitutions in a package by reading the Kptfile.
To print the available substitutions for a package, run `sub` on the package directory
and they will be listed as sub commands.

  PKG_DIR

    A directory containing a Kptfile with substitutions specified.

  SUBSTITUTION_NAME

    The name of the substitution to perform.  To list available substitutions, run
    `sub` against the PKR_DIR with no other arguments.

  NEW_VALUE

    The new value to replace the marker with.

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
      description: 'short description of substitution'

The preceding would enable the command: `kpt sub my-package/ port PORT_NUM`

Substitutions may have the following types: [int, bool, string, float]

### Examples

    # print the substitution commands for a package
    $ kpt sub my-package/
         NAME       REMAINING   PERFORMED         DESCRIPTION          TYPE        MARKER      
      port          4           0           'service port number'     int      $[PORT]         
      name-prefix   4           0           'Resources name prefix'   string   $[NAME_PREFIX]  

    # print help for the port substitution
    $ kpt sub my-package/ port
      NAME   REMAINING   PERFORMED        DESCRIPTION        TYPE   MARKER   
      port   4           0           'service port number'   int    $[PORT]  


    # perform the port substitution in my-package
    $ kpt sub my-package/ port 8080
    performed 4 substitutions