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

.PHONY: cli/fix cli/fmt cli/generate cli/tidy cli/lint cli/test cli/vet cli cli/build

cli/fix:
	make -C cli fix

cli/fmt:
	make -C cli fmt

cli/generate:
	make -C cli generate

cli/tidy:
	make -C cli tidy

cli/lint:
	make -C cli lint

cli/test:
	make -C cli test

cli/vet:
	make -C cli vet

cli:
	make -C cli all

cli/build:
	make -C cli build
