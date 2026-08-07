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

TEST_COVERAGE_FILE=coverage.out
TEST_COVERAGE_HTML_FILE=coverage_unit.html
TEST_COVERAGE_FUNC_FILE=func_coverage.out

##@ Testing

.PHONY: test-coverage
test-coverage: ## Generate coverage reports (runs tests with coverage instrumentation)
	go test -cover -coverprofile=$(TEST_COVERAGE_FILE) ${LDFLAGS} ./...
	go tool cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML_FILE)
	go tool cover -func=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_FUNC_FILE)
	@echo "Coverage reports generated:"
	@echo "  - $(TEST_COVERAGE_FILE): Coverage data (for SonarQube)"
	@echo "  - $(TEST_COVERAGE_HTML_FILE): HTML coverage report"
	@echo "  - $(TEST_COVERAGE_FUNC_FILE): Function-level coverage"

.PHONY: test-clean
test-clean: ## Clean up coverage artifacts
	rm -f $(TEST_COVERAGE_FILE) $(TEST_COVERAGE_HTML_FILE) $(TEST_COVERAGE_FUNC_FILE)
	@echo "Coverage artifacts cleaned"

