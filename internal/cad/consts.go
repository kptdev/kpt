package cad

var kubectlKinds = []string{
	"clusterrole",
	"clusterrolebinding",
	"configmap",
	"cronjob",
	"deployment",
	"ingress",
	"job",
	"namespace",
	"poddisruptionbudget",
	"priorityclass",
	"quota",
	"role",
	"rolebinding",
	"secret",
	"service",
	"serviceaccount",
}

var BuiltinTransformers = map[string]string{
	"namespace": "set-namespace",
}
