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

# name:directory. lib is the root module; its directory is "." so the target
# name is not derived from a folder. mdtogo is intentionally absent.
CHECK_MODULES := lib:. api:api cli:cli
CHECK_TASKS   := fix fmt vet lint test tidy

module_name = $(word 1,$(subst :, ,$(1)))
module_dir  = $(word 2,$(subst :, ,$(1)))

# The root module runs these in place. make -C . fmt would call the aggregate
# fmt target below and recurse.
check_recipe_fix  = go fix ./...
check_recipe_fmt  = go fmt ./...
check_recipe_vet  = go vet ./...
check_recipe_test = go test -cover $(LDFLAGS) ./...
check_recipe_tidy = go mod tidy
check_recipe_lint = @if [[ `command -v golangci-lint` && `golangci-lint version --short` == $(GOLANGCI_LINT_VERSION) ]]; then \
		golangci-lint run -v ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION) run -v ./...; \
	fi

# lib/fmt -> go fmt ./...
# api/fmt -> make -C api fmt
define check_task
.PHONY: $(1)/$(2)
$(1)/$(2):
	$(if $(filter .,$(3)),$(check_recipe_$(2)),$$(MAKE) -C $(3) $(2))
endef

$(foreach spec,$(CHECK_MODULES),$(foreach task,$(CHECK_TASKS), \
  $(eval $(call check_task,$(call module_name,$(spec)),$(task),$(call module_dir,$(spec))))))

# One module at a time. Prerequisite lists would run these in parallel under make -j.
.PHONY: $(CHECK_TASKS)
fix fmt vet lint test tidy:
	$(MAKE) lib/$@
	$(MAKE) api/$@
	$(MAKE) cli/$@

.PHONY: lib api cli
# lib is what the vX.Y.Z tag workflow runs. It does not build the CLI.
lib api cli:
	$(MAKE) $@/fix
	$(MAKE) $@/vet
	$(MAKE) $@/fmt
	$(MAKE) $@/lint
	$(MAKE) $@/test
	$(MAKE) $@/tidy
