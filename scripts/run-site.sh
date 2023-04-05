#!/usr/bin/env bash
# Copyright 2019 The kpt Authors
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

# Set read/execute permissions for newly created site files in macOS or Linux.
setfacl -Rd -m o::rx site/ 2> /dev/null || chmod -R +a "everyone allow read,execute,file_inherit,directory_inherit" site/
# Set read/execute permissions for existing site files.
chmod -R o+rx site/
# Terminate running kpt-site docker containers and rebuild.
docker stop $(docker ps -q --filter ancestor=kpt-site:latest) || docker build site/ -t kpt-site:latest
# Mount the site directory as the default content for the docker container.
docker run -v `pwd`/site:/usr/share/nginx/html -p 3000:80 -d kpt-site:latest
echo "Serving docs at http://127.0.0.1:3000"
