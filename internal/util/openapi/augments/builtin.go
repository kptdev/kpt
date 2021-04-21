package augments

const JsonPatchBuiltin = `
[
  {"op": "add",
    "path": "/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta/properties/annotations/x-kubernetes-field-meaning",
    "value": "annotation"
  },

  {"op": "add",
    "path": "/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta/properties/labels/x-kubernetes-field-meaning",
    "value": "label"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.core.v1.ServiceSpec/properties/selector/x-kubernetes-field-meaning",
    "value": "label"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.core.v1.ReplicationControllerSpec/properties/selector/x-kubernetes-field-meaning",
    "value": "label"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector/properties/matchLabels/x-kubernetes-field-meaning",
    "value": "label"
  },

  {"op": "add",
    "path": "/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.rbac.v1.Subject/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.rbac.v1beta1.Subject/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.admissionregistration.v1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.admissionregistration.v1beta1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.kube-aggregator.pkg.apis.apiregistration.v1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },
  {"op": "add",
    "path": "/definitions/io.k8s.kube-aggregator.pkg.apis.apiregistration.v1beta1.ServiceReference/properties/namespace/x-kubernetes-field-meaning",
    "value": "namespace"
  },

  {"op": "add",
    "path": "/definitions/io.k8s.api.autoscaling.v1.CrossVersionObjectReference/x-kubernetes-object-reference",
    "value": {
      "apiVersion": {"fromField": "apiVersion"},
      "kind": {"fromField": "kind"},
      "name": {"fromField": "name"}
    }
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.autoscaling.v2beta1.CrossVersionObjectReference/x-kubernetes-object-reference",
    "value": {
      "apiVersion": {"fromField": "apiVersion"},
      "kind": {"fromField": "kind"},
      "name": {"fromField": "name"}
    }
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.autoscaling.v2beta2.CrossVersionObjectReference/x-kubernetes-object-reference",
    "value": {
      "apiVersion": {"fromField": "apiVersion"},
      "kind": {"fromField": "kind"},
      "name": {"fromField": "name"}
    }
  },

  {"op": "add",
    "path": "/definitions/io.k8s.api.core.v1.ConfigMapVolumeSource/x-kubernetes-object-reference",
    "value": {
      "apiVersion": {"hardcoded": "v1"},
      "kind": {"hardcoded": "ConfigMap"},
      "name": {"fromField": "name"}
    }
  },
  {"op": "add",
    "path": "/definitions/io.k8s.api.core.v1.ConfigMapKeySelector/x-kubernetes-object-reference",
    "value": {
      "apiVersion": {"hardcoded": "v1"},
      "kind": {"hardcoded": "ConfigMap"},
      "name": {"fromField": "name"}
    }
  }
]`
