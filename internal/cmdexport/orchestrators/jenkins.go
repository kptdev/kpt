// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orchestrators

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
)

// Jenkins is a single-stage Jenkinsfile that uses `any` agent.
type Jenkins struct {
	Stage JenkinsStage
}

func (p *Jenkins) Init(config *types.PipelineConfig) Pipeline {
	p.Stage.Init(config)

	return p
}

func (p *Jenkins) Generate() (out []byte, err error) {
	templateString := `
pipeline {
    agent any

    stages {
        {{.Stage.Generate | indent 8}}
    }
}
`

	result, err := renderTemplate("pipeline", templateString, p)
	out = []byte(result)

	return
}

// JenkinsStage is a stage of a Jenkinsfile. It consists a series of steps to run.
type JenkinsStage struct {
	Name    string
	Scripts []string
}

func (stage *JenkinsStage) Init(config *types.PipelineConfig) *JenkinsStage {
	stage.Name = "Run kpt functions"
	stage.Scripts = []string{
		(&JenkinsStageStep{}).Init(config).Generate(),
	}

	return stage
}

func (stage *JenkinsStage) Generate() (result string, err error) {
	templateString := `
stage('{{.Name}}') {
    steps {
        // This requires that docker is installed on the agent.
        // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
        {{range $i, $script := .Scripts}}sh '''
            {{indent 12 $script}}
        '''{{end}}
    }
}`

	result, err = renderTemplate("stage", templateString, stage)

	return
}

// JenkinsStageStep represents a shell script to execute in a stage in a Jenkinsfile.
type JenkinsStageStep struct {
	MountedWorkspace string
	Dir              string
	FnPaths          []string
}

func (step *JenkinsStageStep) Init(config *types.PipelineConfig) *JenkinsStageStep {
	step.MountedWorkspace = "/app"
	step.Dir = config.Dir
	step.FnPaths = config.FnPaths

	return step
}

func (step *JenkinsStageStep) Generate() string {
	var fnPaths []string
	for _, fnPath := range step.FnPaths {
		fnPath = fmt.Sprintf(
			"--fn-path %s",
			path.Join(step.MountedWorkspace, fnPath),
		)

		fnPaths = append(fnPaths, fnPath)
	}

	multilineScript := &JenkinsMultilineScript{lines: []string{
		"docker run",
		fmt.Sprintf("-v $PWD:%s", step.MountedWorkspace),
		"-v /var/run/docker.sock:/var/run/docker.sock",
		KptImage,
		fmt.Sprintf("fn run %s", path.Join(step.MountedWorkspace, step.Dir)),
	}}
	multilineScript.lines = append(multilineScript.lines, fnPaths...)

	return multilineScript.Generate()
}

// JenkinsMultilineScript represents a multiline script that can be joined using `\`.
type JenkinsMultilineScript struct {
	lines []string
}

// Generate produces a multiline script.
func (script *JenkinsMultilineScript) Generate() string {
	return strings.Join(script.lines, " \\\n")
}

// indent adds spaces to the beginning of each line in the multiline string.
func indent(spaces int, multilineString string) string {
	indentation := strings.Repeat(" ", spaces)
	replacement := fmt.Sprintf("\n%s", indentation)

	return strings.ReplaceAll(
		multilineString,
		"\n",
		replacement,
	)
}

// renderTemplate fills a template from templateString with data.
// Starting empty new lines in the template will be trimmed.
// The indent function is supported in the template.
func renderTemplate(
	templateName string,
	templateString string,
	data interface{},
) (result string, err error) {
	templateString = strings.TrimLeft(templateString, "\n")

	t, err := template.New(templateName).Funcs(template.FuncMap{
		"indent": indent,
	}).Parse(templateString)

	if err != nil {
		return
	}

	b := &bytes.Buffer{}
	err = t.Execute(b, data)
	result = b.String()

	return
}
