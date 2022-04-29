### How to enable jaeger tracing

If you want to enable jaeger tracing of the porch-server:

* Apply the [deployment.yaml manifest](deployment.yaml) from this directory

```
kubectl apply -f deployment.yaml
```

* Add the commented out env var OTEL to the porch-server manifest:

```
kubectl edit deployment -n porch-system porch-server
```

```
        env:
          # Uncomment to enable trace-reporting to jaeger
          #- name: OTEL
          #  value: otel://jaeger-oltp:4317
```

* Port-forward the jaeger http port to your local machine:

```
kubectl port-forward -n porch-system service/jaeger-http 16686
```

* Open your browser to the UI on http://localhost:16686