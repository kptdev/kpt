# Copyright 2026 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Testing tools and targets for SonarQube coverage generation

TEST_COVERAGE_FILE             = coverage.out
TEST_MODULE_COVERAGE_FILE      = module_coverage.out
TEST_MODULE_COVERAGE_HTML_FILE = module_coverage_unit.html
TEST_MODULE_COVERAGE_FUNC_FILE = module_func_coverage.out

##@ Testing

MODULES = $(CURDIR) $(CURDIR)/api $(CURDIR)/mdtogo

MODULE_COVERAGE_FILES      = $(addsuffix /$(TEST_MODULE_COVERAGE_FILE),      $(MODULES))
MODULE_COVERAGE_HTML_FILES = $(addsuffix /$(TEST_MODULE_COVERAGE_HTML_FILE), $(MODULES))
MODULE_COVERAGE_FUNC_FILES = $(addsuffix /$(TEST_MODULE_COVERAGE_FUNC_FILE), $(MODULES))

## Generate per-module and aggregated coverage reports
.PHONY: test-coverage
test-coverage: $(MODULE_COVERAGE_FILES) $(MODULE_COVERAGE_HTML_FILES) $(MODULE_COVERAGE_FUNC_FILES) $(TEST_COVERAGE_FILE) ## Generate coverage reports (runs tests with coverage instrumentation)

.PHONY: FORCE
FORCE:

%/$(TEST_MODULE_COVERAGE_FILE): FORCE
	cd $(dir $@) && go test -cover -coverprofile=$@ ${LDFLAGS} ./...
	@echo "  - $@: Coverage data (for SonarQube)"

%/$(TEST_MODULE_COVERAGE_HTML_FILE): %/$(TEST_MODULE_COVERAGE_FILE)
	cd $(dir $@) && go tool cover -html=$(TEST_MODULE_COVERAGE_FILE) -o $(notdir $@)
	@echo "  - $@: HTML coverage report"

%/$(TEST_MODULE_COVERAGE_FUNC_FILE): %/$(TEST_MODULE_COVERAGE_FILE)
	cd $(dir $@) && go tool cover -func=$(TEST_MODULE_COVERAGE_FILE) -o $(notdir $@)
	@echo "  - $@: Function-level coverage"

$(TEST_COVERAGE_FILE): $(MODULE_COVERAGE_FILES)
	rm -f $@
	@# Merge per-module coverage files; keep the mode line only from the first file.
	head -1 $(firstword $(MODULE_COVERAGE_FILES)) > $@
	for f in $(MODULE_COVERAGE_FILES); do grep -v '^mode:' $$f >> $@; done
	@echo "  - $@: Aggregated coverage data (for SonarQube)"

.PHONY: test-coverage-clean
test-coverage-clean: ## Clean up coverage artifacts
	rm -f $(TEST_COVERAGE_FILE)
	find . -name $(TEST_MODULE_COVERAGE_FILE)      -exec rm -f {} +
	find . -name $(TEST_MODULE_COVERAGE_HTML_FILE) -exec rm -f {} +
	find . -name $(TEST_MODULE_COVERAGE_FUNC_FILE) -exec rm -f {} +
	@echo "Coverage artifacts cleaned"
