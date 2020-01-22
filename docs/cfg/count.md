## count

Count Resources Config from a local directory.

![alt text][tutorial]

    kpt tutorial cfg count

[tutorial-script]

### Synopsis

    kpt cfg count [DIR]

  DIR:
    Path to local directory.

### Examples

    # print Resource counts from a directory
    kpt cfg count my-dir/

    # print Resource counts from a cluster
    kubectl get all -o yaml | kpt cfg count

### 

[tutorial]: https://storage.googleapis.com/kpt-dev/docs/cfg-count.gif "kpt cfg count"
[tutorial-script]: ../gifs/cfg-count.sh
