package get

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func AddGcloudConfigMap(dir, pkgName string) error {
	const op errors.Op = "kptfileutil.PullLocalGcloudConfig"
	data, err := pullLocalGcloudConfig(pkgName)
	if err != nil {
		return err
	}
	cm := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: 
data: {}
`)
	cm.SetName(pkgName)
	cm.SetDataMap(data)
	return writeFile(filepath.Join(dir, pkgutil.LocalGloudConfigFileName(pkgName)), cm)
}

func pullLocalGcloudConfig(pkgName string) (map[string]string, error) {
	projectID := getGcloudConfig("project")
	if projectID == "" {
		return nil, fmt.Errorf("`project` has not been set in `gcloud`")
	}
	account := getGcloudConfig("core/account")
	if account == "" {
		fmt.Println("`core/account` has not been set in `gcloud`")
	}
	zone := getGcloudConfig("compute/zone")
	if zone == "" {
		fmt.Println("`compute/zone` has not been set in `gcloud`")
	}
	region := getGcloudConfig("compute/region")
	if region == "" {
		fmt.Println(fmt.Errorf("`compute/region` has not been set in `gcloud`"))
	}
	orgID, err := getGcloudOrgID(projectID)
	if err != nil {
		return nil, err
	}
	if orgID == "" {
		fmt.Println("`Organization` or `Folder` not found")
	}

	return map[string]string{
		"name":      pkgName,
		"namespace": projectID,
		"projectID": projectID,
		"zone":      zone,
		"region":    region,
		"domain":    account,
		"orgID":     orgID,
	}, nil

}

func getGcloudConfig(property string) string {
	var cmdOut, cmdErr bytes.Buffer
	cmd := exec.Command("gcloud", "config", "get-value", property)
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	err := cmd.Run()
	if err != nil {
		panic(fmt.Errorf("unable to run `gcloud` %v", err.Error()))
	}
	if cmdErr.Len() > 0 {
		return ""
	}
	raw := cmdOut.String()
	return strings.TrimSpace(raw)
}

func getGcloudOrgID(projectID string) (string, error) {
	var buf, err, out bytes.Buffer
	cmdListAncestors := exec.Command("gcloud", "projects", "get-ancestors",
		projectID, "--format=get(id)")
	cmdListAncestors.Stdout = &buf
	cmdListAncestors.Stderr = &err
	cmdListAncestors.Run()
	if err.Len() > 0 {
		return "", fmt.Errorf(err.String())
	}
	cmdOrgID := exec.Command("tail", "-1")
	cmdOrgID.Stdin = &buf
	cmdListAncestors.Stderr = &err
	cmdOrgID.Stdout = &out
	cmdOrgID.Run()
	if err.Len() > 0 {
		return "", fmt.Errorf(err.String())
	}
	raw := out.String()
	return strings.TrimSpace(raw), nil
}

func writeFile(fpath string, cm *yaml.RNode) error {
	const op errors.Op = "kptfileutil.WriteGcloudConfigMapFile"
	out, err := cm.String()
	if err != nil {
		return err
	}
	yaml.MarshalWithOptions(cm.YNode().Value, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	return ioutil.WriteFile(fpath, []byte(out), 0600)

}
