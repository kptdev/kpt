// Copyright 2019 The kpt Authors
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

package cmddocs_test

import (
	"os"
	"path"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/mdtogo/cmddocs"
	"github.com/stretchr/testify/assert"
)

func TestParsingDocWithNameFromFolder(t *testing.T) {
	testDir := path.Join(t.TempDir(), "example")
	dirErr := os.Mkdir(testDir, os.ModePerm)
	assert.NoError(t, dirErr)
	exampleMd, err := os.CreateTemp(testDir, "_index.md")
	assert.NoError(t, err)

	testdata := []byte(`
<!--mdtogo:Short
Short documentation.
-->
Test document.

# Documentation
<!--mdtogo:Long-->
With
long
documentation.
<!--mdtogo-->

# Examples
<!--mdtogo:Examples-->` +
		"```sh\n" +
		`
# An example invocation
example_bin arg1
` +
		"```\n" +
		`

<!--mdtogo-->
	`)

	err = os.WriteFile(exampleMd.Name(), testdata, os.ModePerm)
	assert.NoError(t, err)

	docs := cmddocs.ParseCmdDocs([]string{exampleMd.Name()})
	assert.Equal(t, 1, len(docs))
	assert.Equal(t, "Example", docs[0].Name)
	assert.Equal(t, "Short documentation.", docs[0].Short)
	assert.Equal(t, "\nWith\nlong\ndocumentation.\n", docs[0].Long)
	assert.Equal(t, "\n  \n  # An example invocation\n  example_bin arg1\n", docs[0].Examples)
}

func TestParsingDocWithBackticks(t *testing.T) {
	testDir := path.Join(t.TempDir(), "example")
	dirErr := os.Mkdir(testDir, os.ModePerm)
	assert.NoError(t, dirErr)
	exampleMd, err := os.CreateTemp(testDir, "_index.md")
	assert.NoError(t, err)

	testdata := []byte(`
<!--mdtogo:Short
Short ` +
		"`documentation`" +
		`.
-->
Test document.
` +
		"```\n" +
		`

<!--mdtogo-->
	`)

	err = os.WriteFile(exampleMd.Name(), testdata, os.ModePerm)
	assert.NoError(t, err)

	docs := cmddocs.ParseCmdDocs([]string{exampleMd.Name()})
	assert.Equal(t, 1, len(docs))
	assert.Equal(t, "Example", docs[0].Name)
	assert.Equal(t, "Short ` + \"`\" + `documentation` + \"`\" + `.", docs[0].Short)
	assert.Equal(t, "var ExampleShort = `Short ` + \"`\" + `documentation` + \"`\" + `.`\n", docs[0].String())
}

func TestParsingDocWithNameFromComment(t *testing.T) {
	testDir := path.Join(t.TempDir(), "example")
	dirErr := os.Mkdir(testDir, os.ModePerm)
	assert.NoError(t, dirErr)
	exampleMd, err := os.CreateTemp(testDir, "_index.md")
	assert.NoError(t, err)

	testdata := []byte(`
<!--mdtogo:FirstShort
First short documentation.
-->
Test document.

# Documentation
<!--mdtogo:SecondShort
Second short documentation.
-->
<!--mdtogo:SecondLong-->
With
long
documentation.
<!--mdtogo-->

# Examples
<!--mdtogo:firstExamples-->` +
		"```sh\n" +
		`
# An example invocation
example_bin arg1
` +
		"```\n" +
		`

<!--mdtogo-->
	`)

	err = os.WriteFile(exampleMd.Name(), testdata, os.ModePerm)
	assert.NoError(t, err)

	docs := cmddocs.ParseCmdDocs([]string{exampleMd.Name()})
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	assert.Equal(t, 2, len(docs))

	assert.Equal(t, "First", docs[0].Name)
	assert.Equal(t, "First short documentation.", docs[0].Short)
	assert.Equal(t, "\n  \n  # An example invocation\n  example_bin arg1\n", docs[0].Examples)

	assert.Equal(t, "Second", docs[1].Name)
	assert.Equal(t, "Second short documentation.", docs[1].Short)
	assert.Equal(t, "\nWith\nlong\ndocumentation.\n", docs[1].Long)
}

func TestParsingMultipleDocsFromSameFolder(t *testing.T) {
	testDir := path.Join(t.TempDir(), "example")
	dirErr := os.Mkdir(testDir, os.ModePerm)
	assert.NoError(t, dirErr)
	firstExampleMd, err := os.CreateTemp(testDir, "first_index.md")
	assert.NoError(t, err)
	secondExampleMd, err := os.CreateTemp(testDir, "second_index.md")
	assert.NoError(t, err)

	firstTestData := []byte(`
<!--mdtogo:FirstShort
First short documentation.
-->
Test document.

# Examples
<!--mdtogo:firstExamples-->` +
		"```sh\n" +
		`
# An example invocation
example_bin arg1
` +
		"```\n" +
		`

<!--mdtogo-->
	`)

	secondTestData := []byte(`
Test document.

# Documentation
<!--mdtogo:SecondShort
Second short documentation.
-->
<!--mdtogo:SecondLong-->
With
long
documentation.
<!--mdtogo-->
	`)

	err = os.WriteFile(firstExampleMd.Name(), firstTestData, os.ModePerm)
	assert.NoError(t, err)
	err = os.WriteFile(secondExampleMd.Name(), secondTestData, os.ModePerm)
	assert.NoError(t, err)

	docs := cmddocs.ParseCmdDocs([]string{firstExampleMd.Name(), secondExampleMd.Name()})
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	assert.Equal(t, 2, len(docs))

	assert.Equal(t, "First", docs[0].Name)
	assert.Equal(t, "First short documentation.", docs[0].Short)
	assert.Equal(t, "\n  \n  # An example invocation\n  example_bin arg1\n", docs[0].Examples)

	assert.Equal(t, "Second", docs[1].Name)
	assert.Equal(t, "Second short documentation.", docs[1].Short)
	assert.Equal(t, "\nWith\nlong\ndocumentation.\n", docs[1].Long)
}
