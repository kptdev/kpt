package cad

const PlaceHolder = "example"

var K8sResources = map[string][]string{
	"clusterrole":         nil,
	"clusterrolebinding":  nil,
	"configmap":           nil,
	"cronjob":             nil,
	"deployment":          nil,
	"ingress":             nil,
	"job":                 nil,
	"namespace":           nil,
	"poddisruptionbudget": nil,
	"priorityclass":       nil,
	"quota":               []string{"--hard=cpu=1,memory=1G"},
	"role":                nil,
	"rolebinding":         []string{"--clusterrole=admin", "--group=example-admins@example.com"},
	"secret":              nil,
	"service":             nil,
	"serviceaccount":      nil,
}

func ResourceKinds() []string {
	var kinds []string
	for k, _ := range K8sResources {
		kinds = append(kinds, k)
	}
	return kinds
}

var ResourceContextMap = map[string][]string{
	"namespace": []string{"rolebinding", "quota"},
}

func ResourceKindArgs(kind string) []string {
	flags, ok := K8sResources[kind]
	if !ok {
		return nil
	}
	return flags
}
