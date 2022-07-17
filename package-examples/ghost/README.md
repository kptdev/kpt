```bash
# Fetch ghost helm charts
helm fetch --untar bitnami/ghost
# By default networking, metrics are disabled. 
helm template example ghost > rendered.yaml
rm -rf ghost
# Restructure the KRM resources by app. This should give two directories: ./mariadb /ghost 
kubectl-slice -f rendered.yaml --template '{{ index "app.kubernetes.io/name" .metadata.labels }}/{{.kind | lower}}-{{.metadata.name|dottodash}}.yaml' -o ghost
rm rendered.yaml
```
