// Copyright 2021,2026 The kpt Authors
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

package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	fnruntime "github.com/kptdev/kpt/pkg/fn/runtime"
)

// Runner runs an e2e test
type Runner struct {
	pkgName       string
	testCase      TestCase
	cmd           string
	t             *testing.T
	initialCommit string
	kptBin        string
}

func getKptBin() (string, error) {
	p, err := exec.Command("which", "kpt").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot find command 'kpt' in $PATH: %w", err)
	}
	return strings.TrimSpace(string(p)), nil
}

const (
	// If this env is set to "true", this e2e test framework will update the
	// expected diff and results if they already exist. If will not change
	// config.yaml.
	updateExpectedEnv string = "KPT_E2E_UPDATE_EXPECTED"

	expectedDir         string = ".expected"
	expectedResultsFile string = "results.yaml"
	expectedDiffFile    string = "diff.patch"
	expectedConfigFile  string = "config.yaml"
	outDir              string = "out"
	setupScript         string = "setup.sh"
	teardownScript      string = "teardown.sh"
	execScript          string = "exec.sh"
	CommandFnEval       string = "eval"
	CommandFnRender     string = "render"

	allowWasmFlag string = "--allow-alpha-wasm"
)

// NewRunner returns a new runner for pkg
func NewRunner(t *testing.T, testCase TestCase, c string) (*Runner, error) {
	info, err := os.Stat(testCase.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot open path %s: %w", testCase.Path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", testCase.Path)
	}
	kptBin, err := getKptBin()
	if err != nil {
		t.Logf("failed to find kpt binary: %v", err)
	}
	if kptBin != "" {
		t.Logf("Using kpt binary: %s", kptBin)
	}
	return &Runner{
		pkgName:  filepath.Base(testCase.Path),
		testCase: testCase,
		cmd:      c,
		t:        t,
		kptBin:   kptBin,
	}, nil
}

// Run runs the test.
func (r *Runner) Run() error {
	switch r.cmd {
	case CommandFnEval:
		return r.runFnEval()
	case CommandFnRender:
		return r.runFnRender()
	default:
		return fmt.Errorf("invalid command %s", r.cmd)
	}
}

// runSetupScript runs the setup script if the test has it
func (r *Runner) runSetupScript(pkgPath string) error {
	p, err := filepath.Abs(filepath.Join(r.testCase.Path, expectedDir, setupScript))
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	cmd := getCommand(pkgPath, "bash", []string{p})
	r.t.Logf("running setup script: %q", cmd.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run setup script %q.\nOutput: %q\n: %w", p, string(output), err)
	}
	return nil
}

// runTearDownScript runs the teardown script if the test has it
func (r *Runner) runTearDownScript(pkgPath string) error {
	p, err := filepath.Abs(filepath.Join(r.testCase.Path, expectedDir, teardownScript))
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	cmd := getCommand(pkgPath, "bash", []string{p})
	r.t.Logf("running teardown script: %q", cmd.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run teardown script %q.\nOutput: %q\n: %w", p, string(output), err)
	}
	return nil
}

func (r *Runner) runFnEval() error {
	// run function
	for i := 0; i < r.testCase.Config.RunCount(); i++ {
		r.t.Logf("Running test against package %s, iteration %d \n", r.pkgName, i+1)
		tmpDir, err := os.MkdirTemp("", "krm-fn-e2e-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary dir: %w", err)
		}
		pkgPath := filepath.Join(tmpDir, r.pkgName)

		if r.testCase.Config.Debug {
			fmt.Printf("Running test against package %s in dir %s \n", r.pkgName, pkgPath)
		}
		var resultsDir, destDir string

		if r.IsFnResultExpected() {
			resultsDir = filepath.Join(tmpDir, "results")
		}

		if r.IsOutOfPlace() {
			destDir = filepath.Join(pkgPath, outDir)
		}

		// copy package to temp directory
		err = copyDir(r.testCase.Path, pkgPath)
		if err != nil {
			return fmt.Errorf("failed to copy package: %w", err)
		}

		// init and commit package files
		err = r.preparePackage(pkgPath)
		if err != nil {
			return fmt.Errorf("failed to prepare package: %w", err)
		}

		err = r.runSetupScript(pkgPath)
		if err != nil {
			return err
		}

		var cmd *exec.Cmd
		execScriptPath, err := filepath.Abs(filepath.Join(r.testCase.Path, expectedDir, execScript))
		if err != nil {
			return err
		}

		if _, err := os.Stat(execScriptPath); err == nil {
			cmd = getCommand(pkgPath, "bash", []string{execScriptPath})
		} else {
			kptArgs := []string{"fn", "eval", pkgPath}

			if resultsDir != "" {
				kptArgs = append(kptArgs, "--results-dir", resultsDir)
			}
			if destDir != "" {
				kptArgs = append(kptArgs, "-o", destDir)
			}
			if r.testCase.Config.AllowWasm {
				kptArgs = append(kptArgs, allowWasmFlag)
			}
			if r.testCase.Config.ImagePullPolicy != "" {
				kptArgs = append(kptArgs, "--image-pull-policy", r.testCase.Config.ImagePullPolicy)
			}
			if r.testCase.Config.EvalConfig.Network {
				kptArgs = append(kptArgs, "--network")
			}
			if r.testCase.Config.EvalConfig.Image != "" {
				kptArgs = append(kptArgs, "--image", r.testCase.Config.EvalConfig.Image)
			} else if !r.testCase.Config.EvalConfig.execUniquePath.Empty() {
				kptArgs = append(kptArgs, "--exec", string(r.testCase.Config.EvalConfig.execUniquePath))
			}
			if r.testCase.Config.EvalConfig.Tag != "" {
				kptArgs = append(kptArgs, "--tag", r.testCase.Config.EvalConfig.Tag)
			}
			if !r.testCase.Config.EvalConfig.fnConfigUniquePath.Empty() {
				kptArgs = append(kptArgs, "--fn-config", string(r.testCase.Config.EvalConfig.fnConfigUniquePath))
			}
			if r.testCase.Config.EvalConfig.IncludeMetaResources {
				kptArgs = append(kptArgs, "--include-meta-resources")
			}
			// args must be appended last
			if len(r.testCase.Config.EvalConfig.Args) > 0 {
				kptArgs = append(kptArgs, "--")
				for k, v := range r.testCase.Config.EvalConfig.Args {
					kptArgs = append(kptArgs, fmt.Sprintf("%s=%s", k, v))
				}
			}
			cmd = getCommand("", r.kptBin, kptArgs)
		}
		r.t.Logf("running command: %v=%v %v", fnruntime.ContainerRuntimeEnv, os.Getenv(fnruntime.ContainerRuntimeEnv), cmd.String())
		stdout, stderr, fnErr := runCommand(cmd)
		if fnErr != nil {
			r.t.Logf("kpt error, stdout: %s; stderr: %s", stdout, stderr)
		}
		// Update the diff file or results file if updateExpectedEnv is set.
		if strings.ToLower(os.Getenv(updateExpectedEnv)) == "true" {
			return r.updateExpected(pkgPath, resultsDir, filepath.Join(r.testCase.Path, expectedDir))
		}

		// compare results
		err = r.compareResult(fnErr, stdout, sanitizeTimestamps(stderr), pkgPath, resultsDir)
		if err != nil {
			return err
		}
		// we passed result check, now we should break if the command error
		// is expected
		if fnErr != nil {
			break
		}

		err = r.runTearDownScript(pkgPath)
		if err != nil {
			return err
		}

		// cleanup temp directory after iteration
		if !r.testCase.Config.Debug {
			os.RemoveAll(tmpDir)
		}
	}

	return nil
}

func sanitizeTimestamps(stderr string) string {
	// Output will have non-deterministic output timestamps. We will replace these to static message for
	// stable comparison in tests.
	var sanitized []string
	for line := range strings.SplitSeq(stderr, "\n") {
		// [PASS] \"ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest\" in 2.0s
		if strings.HasPrefix(line, "[PASS]") || strings.HasPrefix(line, "[FAIL]") {
			tokens := strings.Fields(line)
			if len(tokens) == 4 && tokens[2] == "in" {
				tokens[3] = "0s"
				line = strings.Join(tokens, " ")
			}
		}
		sanitized = append(sanitized, line)
	}
	return strings.Join(sanitized, "\n")
}

// IsFnResultExpected determines if function results are expected for this testcase.
func (r *Runner) IsFnResultExpected() bool {
	_, err := os.ReadFile(filepath.Join(r.testCase.Path, expectedDir, expectedResultsFile))
	return err == nil
}

// IsOutOfPlace determines if command output is saved in a different directory (out-of-place).
func (r *Runner) IsOutOfPlace() bool {
	_, err := os.ReadDir(filepath.Join(r.testCase.Path, outDir))
	return err == nil
}

func (r *Runner) runFnRender() error {
	// run function
	for i := 0; i < r.testCase.Config.RunCount(); i++ {
		r.t.Logf("Running test against package %s, iteration %d \n", r.pkgName, i+1)
		tmpDir, err := os.MkdirTemp("", "kpt-pipeline-e2e-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary dir: %w", err)
		}
		if r.testCase.Config.Debug {
			fmt.Printf("Running test against package %s in dir %s \n", r.pkgName, tmpDir)
		}
		pkgPath := filepath.Join(tmpDir, r.pkgName)
		// create dir to store untouched pkg to compare against
		origPkgPath := filepath.Join(tmpDir, "original")
		err = os.Mkdir(origPkgPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create original dir %s: %w", origPkgPath, err)
		}

		var resultsDir, destDir string

		if r.IsFnResultExpected() {
			resultsDir = filepath.Join(tmpDir, "results")
		}

		if r.IsOutOfPlace() {
			destDir = filepath.Join(pkgPath, outDir)
		}

		// copy package to temp directory
		err = copyDir(r.testCase.Path, pkgPath)
		if err != nil {
			return fmt.Errorf("failed to copy package: %w", err)
		}
		err = copyDir(r.testCase.Path, origPkgPath)
		if err != nil {
			return fmt.Errorf("failed to copy package: %w", err)
		}

		// init and commit package files
		err = r.preparePackage(pkgPath)
		if err != nil {
			return fmt.Errorf("failed to prepare package: %w", err)
		}

		err = r.runSetupScript(pkgPath)
		if err != nil {
			return err
		}

		var cmd *exec.Cmd

		execScriptPath, err := filepath.Abs(filepath.Join(r.testCase.Path, expectedDir, execScript))
		if err != nil {
			return err
		}

		if _, err := os.Stat(execScriptPath); err == nil {
			cmd = getCommand(pkgPath, "bash", []string{execScriptPath})
		} else {
			kptArgs := []string{"fn", "render", pkgPath}

			if resultsDir != "" {
				kptArgs = append(kptArgs, "--results-dir", resultsDir)
			}

			if destDir != "" {
				kptArgs = append(kptArgs, "-o", destDir)
			}

			if r.testCase.Config.ImagePullPolicy != "" {
				kptArgs = append(kptArgs, "--image-pull-policy", r.testCase.Config.ImagePullPolicy)
			}

			if r.testCase.Config.AllowExec {
				kptArgs = append(kptArgs, "--allow-exec")
			}

			if r.testCase.Config.AllowWasm {
				kptArgs = append(kptArgs, allowWasmFlag)
			}

			if r.testCase.Config.DisableOutputTruncate {
				kptArgs = append(kptArgs, "--truncate-output=false")
			}
			cmd = getCommand("", r.kptBin, kptArgs)
		}
		r.t.Logf("running command: %v=%v %v", fnruntime.ContainerRuntimeEnv, os.Getenv(fnruntime.ContainerRuntimeEnv), cmd.String())
		stdout, stderr, fnErr := runCommand(cmd)
		// Update the diff file or results file if updateExpectedEnv is set.
		if strings.ToLower(os.Getenv(updateExpectedEnv)) == "true" {
			return r.updateExpected(pkgPath, resultsDir, filepath.Join(r.testCase.Path, expectedDir))
		}

		if fnErr != nil {
			r.t.Logf("kpt error, stdout: %s; stderr: %s", stdout, stderr)
		}
		// compare results
		err = r.compareResult(fnErr, stdout, sanitizeTimestamps(stderr), pkgPath, resultsDir)
		if err != nil {
			return err
		}
		// we passed result check, now we should run teardown script and break
		// if the command error is expected
		err = r.runTearDownScript(pkgPath)
		if err != nil {
			return err
		}

		// cleanup temp directory after iteration
		if !r.testCase.Config.Debug {
			os.RemoveAll(tmpDir)
		}

		if fnErr != nil {
			break
		}
	}
	return nil
}

func (r *Runner) preparePackage(pkgPath string) error {
	err := gitInit(pkgPath)
	if err != nil {
		return err
	}

	err = gitAddAll(pkgPath)
	if err != nil {
		return err
	}

	err = gitCommit(pkgPath, "first")
	if err != nil {
		return err
	}

	r.initialCommit, err = getCommitHash(pkgPath)
	return err
}

func (r *Runner) compareResult(exitErr error, stdout string, inStderr string, tmpPkgPath, resultsPath string) error {
	stderr := r.stripLines(inStderr, r.testCase.Config.StdErrStripLines)

	expected, err := newExpected(tmpPkgPath)
	if err != nil {
		return err
	}
	// get exit code
	exitCode := 0
	if e, ok := exitErr.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	} else if exitErr != nil {
		return fmt.Errorf("cannot get exit code, received error '%w'", exitErr)
	}

	if exitCode != r.testCase.Config.ExitCode {
		return fmt.Errorf("actual exit code %d doesn't match expected %d", exitCode, r.testCase.Config.ExitCode)
	}

	err = r.compareOutput(stdout, stderr)
	if err != nil {
		return err
	}

	// compare results
	actualResults, err := readActualResults(resultsPath)
	if err != nil {
		return fmt.Errorf("failed to read actual results: %w", err)
	}

	actualResults = r.stripLines(actualResults, r.testCase.Config.ActualStripLines)

	diffOfResult, err := diffStrings(actualResults, expected.Results)
	if err != nil {
		return fmt.Errorf("error when run diff of results: %w: %s", err, diffOfResult)
	}
	if actualResults != expected.Results {
		return fmt.Errorf("actual results doesn't match expected\nActual\n===\n%s\nDiff of Results\n===\n%s",
			actualResults, diffOfResult)
	}

	// compare diff
	actualDiff, err := readActualDiff(tmpPkgPath, r.initialCommit)
	if err != nil {
		return fmt.Errorf("failed to read actual diff: %w", err)
	}
	expectedDiff := expected.Diff
	actualDiff, err = normalizeDiff(actualDiff, r.testCase.Config.DiffStripRegEx)
	if err != nil {
		return err
	}
	expectedDiff, err = normalizeDiff(expectedDiff, r.testCase.Config.DiffStripRegEx)
	if err != nil {
		return err
	}
	if actualDiff != expectedDiff {
		diffOfDiff, err := diffStrings(actualDiff, expectedDiff)
		if err != nil {
			return fmt.Errorf("error when run diff of diff: %w: %s", err, diffOfDiff)
		}
		return fmt.Errorf("actual diff doesn't match expected\nActual\n===\n%s\nDiff of Diff\n===\n%s",
			actualDiff, diffOfDiff)
	}
	return nil
}

// check stdout and stderr against expected
func (r *Runner) compareOutput(stdout string, stderr string) error {
	expectedStderr := r.testCase.Config.StdErr
	conditionedStderr := removeArmPlatformWarning(stderr)

	if !strings.Contains(conditionedStderr, expectedStderr) {
		r.t.Logf("stderr diff is %s", cmp.Diff(expectedStderr, conditionedStderr))
		return fmt.Errorf("wanted stderr %q, got %q", expectedStderr, conditionedStderr)
	}
	stdErrRegEx := r.testCase.Config.StdErrRegEx
	if stdErrRegEx != "" {
		r, err := regexp.Compile(stdErrRegEx)
		if err != nil {
			return fmt.Errorf("unable to compile the regular expression %q: %w", stdErrRegEx, err)
		}
		if !r.MatchString(conditionedStderr) {
			return fmt.Errorf("unable to match regular expression %q, got %v", stdErrRegEx, conditionedStderr)
		}
	}
	expectedStdout := r.testCase.Config.StdOut
	if !strings.Contains(stdout, expectedStdout) {
		r.t.Logf("stdout diff is %s", cmp.Diff(expectedStdout, stdout))
		return fmt.Errorf("wanted stdout %q, got %q", expectedStdout, stdout)
	}
	return nil
}

func (r *Runner) Skip() bool {
	return r.testCase.Config.Skip
}

func readActualResults(resultsPath string) (string, error) {
	// no results
	if resultsPath == "" {
		return "", nil
	}
	l, err := os.ReadDir(resultsPath)
	if err != nil {
		return "", fmt.Errorf("failed to get files in results dir: %w", err)
	}
	if len(l) > 1 {
		return "", fmt.Errorf("unexpected results files number %d, should be 0 or 1", len(l))
	}
	if len(l) == 0 {
		// no result file
		return "", nil
	}
	resultsFile := l[0].Name()
	actualResults, err := os.ReadFile(filepath.Join(resultsPath, resultsFile))
	if err != nil {
		return "", fmt.Errorf("failed to read actual results: %w", err)
	}
	return strings.TrimSpace(string(actualResults)), nil
}

func readActualDiff(path, origHash string) (string, error) {
	err := gitAddAll(path)
	if err != nil {
		return "", err
	}
	err = gitCommit(path, "second")
	if err != nil {
		return "", err
	}
	// diff with first commit
	actualDiff, err := gitDiff(path, origHash, "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(actualDiff), nil
}

// expected contains the expected result for the function running
type expected struct {
	Results string
	Diff    string
}

func newExpected(path string) (expected, error) {
	e := expected{}
	// get expected results
	expectedResults, err := os.ReadFile(filepath.Join(path, expectedDir, expectedResultsFile))
	switch {
	case os.IsNotExist(err):
		e.Results = ""
	case err != nil:
		return e, fmt.Errorf("failed to read expected results: %w", err)
	default:
		e.Results = strings.TrimSpace(string(expectedResults))
	}

	// get expected diff
	expectedDiff, err := os.ReadFile(filepath.Join(path, expectedDir, expectedDiffFile))
	switch {
	case os.IsNotExist(err):
		e.Diff = ""
	case err != nil:
		return e, fmt.Errorf("failed to read expected diff: %w", err)
	default:
		e.Diff = strings.TrimSpace(string(expectedDiff))
	}

	return e, nil
}

func (r *Runner) updateExpected(tmpPkgPath, resultsPath, sourceOfTruthPath string) error {
	if resultsPath != "" {
		// We update results directory only when a result file already exists.
		l, err := os.ReadDir(resultsPath)
		if err != nil {
			return err
		}
		if len(l) > 0 {
			actualResults, err := readActualResults(resultsPath)
			if err != nil {
				return err
			}
			if actualResults != "" {
				if err := os.WriteFile(filepath.Join(sourceOfTruthPath, expectedResultsFile), []byte(actualResults+"\n"), 0666); err != nil {
					return err
				}
			}
		}
	}
	actualDiff, err := readActualDiff(tmpPkgPath, r.initialCommit)
	if err != nil {
		return err
	}
	if actualDiff != "" {
		if err := os.WriteFile(filepath.Join(sourceOfTruthPath, expectedDiffFile), []byte(actualDiff+"\n"), 0666); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) stripLines(string2Strip string, linesToStrip []string) string {
	strippedString := string2Strip

	for _, line2Strip := range linesToStrip {
		strippedString = strings.ReplaceAll(strippedString, line2Strip+"\n", "")
	}

	return strippedString
}

// normalizeDiff removes lines matching stripRegEx and normalizes index/hunk
// headers in the diff string so that environment-specific output does not
// cause comparison failures.
func normalizeDiff(diff, stripRegEx string) (string, error) {
	var re *regexp.Regexp
	var err error
	if stripRegEx != "" {
		re, err = regexp.Compile(stripRegEx)
		if err != nil {
			return "", fmt.Errorf("unable to compile DiffStripRegEx %q: %w", stripRegEx, err)
		}
	}
	// Normalize CRLF to LF for cross-platform safety.
	diff = strings.ReplaceAll(diff, "\r", "")
	indexRE := regexp.MustCompile(`^index [0-9a-f]+\.\.[0-9a-f]+`)
	hunkRE := regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@.*$`)
	doubleQuotedScalarRE := regexp.MustCompile(`^(\s*-?\s*[^:]+:\s*)"(.*)"\s*$`)
	singleQuotedScalarRE := regexp.MustCompile(`^(\s*-?\s*[^:]+:\s*)'(.*)'\s*$`)
	mapKeyOnlyRE := regexp.MustCompile(`^[A-Za-z0-9_.-]+:\s*.*$`)

	n := &diffNormalizer{
		re:                   re,
		indexRE:              indexRE,
		hunkRE:               hunkRE,
		doubleQuotedScalarRE: doubleQuotedScalarRE,
		singleQuotedScalarRE: singleQuotedScalarRE,
		mapKeyOnlyRE:         mapKeyOnlyRE,
	}
	return n.normalize(diff), nil
}

type diffNormalizer struct {
	re                   *regexp.Regexp
	indexRE              *regexp.Regexp
	hunkRE               *regexp.Regexp
	doubleQuotedScalarRE *regexp.Regexp
	singleQuotedScalarRE *regexp.Regexp
	mapKeyOnlyRE         *regexp.Regexp

	out           []string
	kptChangedRun []string
	inKptfileDiff bool
}

func (n *diffNormalizer) isKptfileDiffHeader(line string) bool {
	parts := strings.Fields(line)
	if len(parts) < 4 || parts[0] != "diff" || parts[1] != "--git" {
		return false
	}
	left := parts[2]
	right := parts[3]
	leftIsKptfile := left == "a/Kptfile" || strings.HasSuffix(left, "/Kptfile")
	rightIsKptfile := right == "b/Kptfile" || strings.HasSuffix(right, "/Kptfile")
	return leftIsKptfile && rightIsKptfile
}

func (n *diffNormalizer) normalizePayload(payload string) string {
	payload = strings.TrimLeft(payload, " \t")
	if m := n.doubleQuotedScalarRE.FindStringSubmatch(payload); m != nil {
		payload = m[1] + m[2]
	} else if m := n.singleQuotedScalarRE.FindStringSubmatch(payload); m != nil {
		payload = m[1] + strings.ReplaceAll(m[2], "''", "'")
	}
	return payload
}

func (n *diffNormalizer) isSortableMapKey(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return n.mapKeyOnlyRE.MatchString(trimmed) && !strings.HasPrefix(trimmed, "- ")
}

type block struct {
	lines    []string
	sortable bool
}

func (n *diffNormalizer) sortChunk(chunk []string) {
	if len(chunk) <= 1 {
		return
	}

	var blocks []block
	for i := 0; i < len(chunk); {
		if !n.isSortableMapKey(chunk[i]) {
			blocks = append(blocks, block{lines: []string{chunk[i]}, sortable: false})
			i++
			continue
		}

		j := i + 1
		for j < len(chunk) && !n.isSortableMapKey(chunk[j]) {
			j++
		}
		blocks = append(blocks, block{lines: append([]string(nil), chunk[i:j]...), sortable: true})
		i = j
	}

	for i := 0; i < len(blocks); {
		if !blocks[i].sortable {
			i++
			continue
		}
		j := i + 1
		for j < len(blocks) && blocks[j].sortable {
			j++
		}
		sort.Slice(blocks[i:j], func(a, b int) bool {
			return blocks[i+a].lines[0] < blocks[i+b].lines[0]
		})
		i = j
	}

	idx := 0
	for _, b := range blocks {
		for _, line := range b.lines {
			chunk[idx] = line
			idx++
		}
	}
}

func (n *diffNormalizer) isListItem(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "- image:")
}

func (n *diffNormalizer) sortMapKeySegments(lines []string) {
	for i := 0; i < len(lines); {
		if n.isListItem(lines[i]) {
			j := i + 1
			for j < len(lines) && !n.isListItem(lines[j]) {
				j++
			}
			n.sortChunk(lines[i+1 : j])
			i = j
			continue
		}
		j := i
		for j < len(lines) && !n.isListItem(lines[j]) {
			j++
		}
		n.sortChunk(lines[i:j])
		i = j
	}
}

func (n *diffNormalizer) flushKptChangedRun() {
	if len(n.kptChangedRun) == 0 {
		return
	}

	var removed []string
	var added []string
	for _, runLine := range n.kptChangedRun {
		switch runLine[0] {
		case '-':
			removed = append(removed, runLine[1:])
		case '+':
			added = append(added, runLine[1:])
		}
	}

	for i := range removed {
		removed[i] = n.normalizePayload(removed[i])
	}
	for i := range added {
		added[i] = n.normalizePayload(added[i])
	}

	n.sortMapKeySegments(removed)
	n.sortMapKeySegments(added)

	for _, line := range removed {
		n.out = append(n.out, "-"+line)
	}
	for _, line := range added {
		n.out = append(n.out, "+"+line)
	}
	n.kptChangedRun = nil
}

func (n *diffNormalizer) normalize(diff string) string {
	for _, line := range strings.Split(diff, "\n") {
		if line == `\ No newline at end of file` {
			continue
		}
		if strings.HasPrefix(line, "diff --git ") {
			n.flushKptChangedRun()
			n.inKptfileDiff = n.isKptfileDiffHeader(line)
		}

		if n.inKptfileDiff &&
			(len(line) > 0 && (line[0] == '+' || line[0] == '-')) &&
			!strings.HasPrefix(line, "+++") &&
			!strings.HasPrefix(line, "---") {
			// Ignore indentation-only drift in Kptfile changed lines.
			line = line[:1] + strings.TrimLeft(line[1:], " \t")
			if n.re != nil && n.re.MatchString(line) {
				continue
			}
			line = n.indexRE.ReplaceAllString(line, "index NORMALIZED")
			line = n.hunkRE.ReplaceAllString(line, "@@ NORMALIZED @@")
			n.kptChangedRun = append(n.kptChangedRun, line)
			continue
		}
		if n.inKptfileDiff && strings.HasPrefix(line, " ") {
			// Hunk context lines are unstable anchors; compare only semantic changes.
			continue
		}
		n.flushKptChangedRun()

		if n.re != nil && n.re.MatchString(line) {
			continue
		}
		line = n.indexRE.ReplaceAllString(line, "index NORMALIZED")
		line = n.hunkRE.ReplaceAllString(line, "@@ NORMALIZED @@")
		// Strip leading whitespace from non-Kptfile context/changed lines
		// to make comparison indentation-insensitive across environments.
		if len(line) > 0 && (line[0] == ' ' || line[0] == '+' || line[0] == '-') &&
			!strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") &&
			!strings.HasPrefix(line, "diff --git") {
			line = line[:1] + strings.TrimLeft(line[1:], " \t")
		}
		n.out = append(n.out, line)
	}
	n.flushKptChangedRun()
	return strings.Join(n.out, "\n")
}
