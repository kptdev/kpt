package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/run"
)

// Runner runs an e2e test
type Runner struct {
	pkgName  string
	testCase TestCase
	cmd      string
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
	CommandFnEval       string = "run"
	CommandFnRender     string = "render"
)

// NewRunner returns a new runner for pkg
func NewRunner(testCase TestCase, c string) (*Runner, error) {
	info, err := os.Stat(testCase.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot open path %s: %w", testCase.Path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", testCase.Path)
	}
	return &Runner{
		pkgName:  filepath.Base(testCase.Path),
		testCase: testCase,
		cmd:      c,
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

func (r *Runner) runFnEval() error {
	fmt.Printf("Running test against package %s\n", r.pkgName)
	tmpDir, err := ioutil.TempDir("", "kpt-fn-e2e-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	tmpPkgPath := filepath.Join(tmpDir, r.pkgName)
	// create result dir
	resultsPath := filepath.Join(tmpDir, "results")
	err = os.Mkdir(resultsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create results dir %s: %w", resultsPath, err)
	}

	// copy package to temp directory
	err = copyDir(r.testCase.Path, tmpPkgPath)
	if err != nil {
		return fmt.Errorf("failed to copy package: %w", err)
	}

	// init and commit package files
	err = r.preparePackage(tmpPkgPath)
	if err != nil {
		return fmt.Errorf("failed to prepare package: %w", err)
	}

	// run function
	kptArgs := []string{"fn", "run", tmpPkgPath, "--results-dir", resultsPath}
	if r.testCase.Config.Network {
		kptArgs = append(kptArgs, "--network")
	}
	var output string
	var fnErr error
	for i := 0; i < r.testCase.Config.RunCount; i++ {
		output, fnErr = runCommand("", "kpt", kptArgs)
		if fnErr != nil {
			// if kpt fn run returns error, we should compare
			// the result
			break
		}
	}

	// run formatter
	_, err = runCommand("", "kpt", []string{"cfg", "fmt", tmpPkgPath})
	if err != nil {
		return fmt.Errorf("failed to run kpt cfg fmt: %w", err)
	}

	// Update the diff file or results file if updateExpectedEnv is set.
	if strings.ToLower(os.Getenv(updateExpectedEnv)) == "true" {
		return updateExpected(tmpPkgPath, resultsPath, filepath.Join(r.testCase.Path, expectedDir))
	}

	// compare results
	err = r.compareResult(fnErr, tmpPkgPath, resultsPath)
	if err != nil {
		return fmt.Errorf("%w\nkpt output:\n%s", err, output)
	}
	return nil
}

func (r *Runner) runFnRender() error {
	tmpDir, err := ioutil.TempDir("", "kpt-pipeline-e2e-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}
	if r.testCase.Config.Debug {
		fmt.Printf("Running test against package %s in dir %s \n", r.pkgName, tmpDir)
	}
	if !r.testCase.Config.Debug {
		// if debug is true, keep the test directory around for debugging
		defer os.RemoveAll(tmpDir)
	}
	tmpPkgPath := filepath.Join(tmpDir, r.pkgName)
	// create dir to store untouched pkg to compare against
	orgPkgPath := filepath.Join(tmpDir, "original")
	err = os.Mkdir(orgPkgPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create original dir %s: %w", orgPkgPath, err)
	}

	// copy package to temp directory
	err = copyDir(r.testCase.Path, tmpPkgPath)
	if err != nil {
		return fmt.Errorf("failed to copy package: %w", err)
	}
	err = copyDir(r.testCase.Path, orgPkgPath)
	if err != nil {
		return fmt.Errorf("failed to copy package: %w", err)
	}

	// init and commit package files
	err = r.preparePackage(tmpPkgPath)
	if err != nil {
		return fmt.Errorf("failed to prepare package: %w", err)
	}

	// run function
	var fnErr error
	command := run.GetMain()
	kptArgs := []string{"fn", "render", tmpPkgPath}
	for i := 0; i < r.testCase.Config.RunCount; i++ {
		command.SetArgs(kptArgs)
		fnErr = command.Execute()
		if fnErr != nil {
			if r.testCase.Config.ExitCode != 0 {
				return nil
			}
			break
		}
	}

	// Update the diff file or results file if updateExpectedEnv is set.
	if strings.ToLower(os.Getenv(updateExpectedEnv)) == "true" {
		return updateExpected(tmpPkgPath, orgPkgPath, filepath.Join(r.testCase.Path, expectedDir))
	}

	// compare results
	err = r.compareResult(fnErr, tmpPkgPath, orgPkgPath)
	return err
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

	return gitCommit(pkgPath, "first")
}

func (r *Runner) compareResult(exitErr error, tmpPkgPath, resultsPath string) error {
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

	if exitCode != 0 {
		actual, err := readActualResults(resultsPath)
		if err != nil {
			return fmt.Errorf("failed to read actual results: %w", err)
		}
		if actual != expected.Results {
			return fmt.Errorf("actual results doesn't match expected\nActual\n===\n%s\nExpected\n===\n%s",
				actual, expected.Results)
		}
		return nil
	}

	// compare diff
	actual, err := readActualDiff(tmpPkgPath)
	if err != nil {
		return fmt.Errorf("failed to read actual diff: %w", err)
	}
	if actual != expected.Diff {
		return fmt.Errorf("actual diff doesn't match expected\nActual\n===\n%s\nExpected\n===\n%s",
			actual, expected.Diff)
	}
	return nil
}

func (r *Runner) Skip() bool {
	return r.testCase.Config.Skip
}

func readActualResults(resultsPath string) (string, error) {
	l, err := ioutil.ReadDir(resultsPath)
	if err != nil {
		return "", fmt.Errorf("failed to get files in results dir: %w", err)
	}
	if len(l) != 1 {
		return "", fmt.Errorf("unexpected results files number %d, should be 1", len(l))
	}
	resultsFile := l[0].Name()
	actualResults, err := ioutil.ReadFile(filepath.Join(resultsPath, resultsFile))
	if err != nil {
		return "", fmt.Errorf("failed to read actual results: %w", err)
	}
	return strings.TrimSpace(string(actualResults)), nil
}

func readActualDiff(path string) (string, error) {
	err := gitAddAll(path)
	if err != nil {
		return "", err
	}
	err = gitCommit(path, "second")
	if err != nil {
		return "", err
	}
	// diff with first commit
	actualDiff, err := gitDiff(path, "HEAD^", "HEAD")
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
	expectedResults, err := ioutil.ReadFile(filepath.Join(path, expectedDir, expectedResultsFile))
	switch {
	case os.IsNotExist(err):
		e.Results = ""
	case err != nil:
		return e, fmt.Errorf("failed to read expected results: %w", err)
	default:
		e.Results = strings.TrimSpace(string(expectedResults))
	}

	// get expected diff
	expectedDiff, err := ioutil.ReadFile(filepath.Join(path, expectedDir, expectedDiffFile))
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

func updateExpected(tmpPkgPath, resultsPath, sourceOfTruthPath string) error {
	// We update results directory only when a result file already exists.
	l, err := ioutil.ReadDir(resultsPath)
	if err != nil {
		return err
	}
	if len(l) > 0 {
		actualResults, err := readActualResults(resultsPath)
		if err != nil {
			return err
		}
		if actualResults != "" {
			if err := ioutil.WriteFile(filepath.Join(sourceOfTruthPath, expectedResultsFile), []byte(actualResults+"\n"), 0666); err != nil {
				return err
			}
		}
	} else {
		actualDiff, err := readActualDiff(tmpPkgPath)
		if err != nil {
			return err
		}
		if actualDiff != "" {
			if err := ioutil.WriteFile(filepath.Join(sourceOfTruthPath, expectedDiffFile), []byte(actualDiff+"\n"), 0666); err != nil {
				return err
			}
		}
	}

	return nil
}
