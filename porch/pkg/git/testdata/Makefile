# Copyright 2022 The kpt Authors
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

REPOSITORIES = \
  empty-repository.tar \
  trivial-repository.tar \
  simple-repository.tar \
  drafts-repository.tar \
  nested-repository.tar \
  publishinfo-repository.tar


.PHONY: expand
expand:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  rm -rf $${dir}; mkdir $${dir}; \
	  tar xf $${f} -C $${dir}; \
	  echo "$${f} --> $${dir}/"; \
	done

.PHONY: checkout
checkout: expand
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  (cd $${dir}; git reset --hard refs/heads/main -- 2>/dev/null ); \
	done

.PHONY: tars
tars:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  ../../../scripts/tar-test-repo.sh $${dir}/ $${f}; \
	  rm -rf $${dir}; \
	  echo "$${dir}/ --> $${f}"; \
	done

.PHONY: clean
clean:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  rm -rf $${dir}; \
	done
