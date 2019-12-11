## sub

Substitute values into Resources fields

### Synopsis

Substitute values into Resources fields by replacing substitution markers with
user supplied values.

`sub` looks for substitutions specified in a Kptfile, and replaces the substitution markers
with user provided values.

To print the available substitutions for a package, run `sub PKG_DIR/` without additional
arguments.

  PKG_DIR

    A directory containing a Kptfile.

  SUBSTITUTION_NAME

    Optional.  The name of the substitution to perform or show.

  NEW_VALUE

    Optional.  The new value to replace the marker with.  If no value is
    specified, substitutions will be printed only.

The following is an example Kptfile containing a substitution definition.  The substitution
has the following pieces:

- `name` of the substitution
- `type` of the value that will be substituted
- `marker` that will be replaced with the value
- `paths` to fields that may contain the marker
- `description` of the substitution

It substitutes and integer value for the marker `$[PORT]` at the provided field
paths.

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

Substitutions may have the following types and are checked before substitution is 
performed: [int, bool, string, float]

**Note**: run `kpt sub PKG_DIR/` on package after you fetch them to determine if
any substitutions are required before the package is used.  The command will exit
non-0 if any substitutions are unfulfilled.

Substitutions once performed maybe overridden or reverted with the `--override` and
`--revert` flags (respectively).

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
    
    # override a previous substitution with a new value
    $ kpt sub my-package/ port 8081 --override
    performed 4 substitutions
    
    # revert a previous substitution
    $ kpt sub my-package/ port --revert
    performed 4 substitutions