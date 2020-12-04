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
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"github.com/stretchr/testify/assert"
)

func TestGenerateJenkinsStageStep(t *testing.T) {
	step := &JenkinsStageStep{
		MountedWorkspace: "/app",
		Dir:              "resources",
		FnPaths:          []string{"function1.yaml", "function2.yaml"},
	}

	expected := `
docker run \
-v $PWD:/app \
-v /var/run/docker.sock:/var/run/docker.sock \
gcr.io/kpt-dev/kpt:latest \
fn run /app/resources \
--fn-path /app/function1.yaml \
--fn-path /app/function2.yaml`
	script := step.Generate()

	assert.Equal(t, script, strings.TrimLeft(expected, "\n"))
}

var jenkinsTestCases = []testCase{
	{
		description: "generate a Jenkinsfile",
		config: &types.PipelineConfig{
			Dir: "resources",
		},
		expected: `
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/resources
                '''
            }
        }
    }
}
`,
	},
	{
		description: "generate a Jenkinsfile with --fn-path",
		config: &types.PipelineConfig{
			Dir:     "resources",
			FnPaths: []string{"functions.yaml"},
		},
		expected: `
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/resources \
                    --fn-path /app/functions.yaml
                '''
            }
        }
    }
}
`,
	},
}

var jenkinsTestSuite = testSuite{
	pipeline:  new(Jenkins),
	testCases: jenkinsTestCases,
}
