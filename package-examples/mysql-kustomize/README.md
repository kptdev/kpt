upstream
==================================================

# NAME

  mysql-kustomize

# SYNOPSIS

To apply the package:

    kustomize build mysql-kustomize/instance | kubectl apply -R -f -

To connect to the database:

    kubectl run -t -i mysql-debug --image mysql:5.7.14 bash
    mysql -u root -h mysql-0.mysql -pPASSWORD

To edit the package:

    kpt fn eval --image gcr.io/kpt-fn/apply-setters:VERSION mysql-kustomize --  [setter_name=setter_value]

# Description

The mysql-kustomize package runs a single instance of the mysql database
as a StatefulSet.

# SEE ALSO
