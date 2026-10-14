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

.PHONY: api/fix api/fmt api/generate api/tidy api/lint api/test api/vet api

api/fix:
	make -C api fix

api/fmt:
	make -C api fmt

api/generate:
	make -C api generate

api/tidy:
	make -C api tidy

api/lint:
	make -C api lint

api/test:
	make -C api test

api/vet:
	make -C api vet

api:
	make -C api all
