# E-commerce Example (kpt)

## Overview
A simple example showing how Kubernetes applications can be managed using kpt. Users can customize the app via configuration files instead of changing code.

## Structure
- `deployment.yaml` → Application Deployment
- `service.yaml` → Service to expose the app
- `Kptfile` → kpt package metadata

## Deployment
Apply the Kubernetes resources:

```bash
kubectl apply -f deployment.yaml -f service.yaml
