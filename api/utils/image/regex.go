// Copyright 2020 The regclient Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// taken from https://github.com/regclient/regclient/blob/b413945eb6f65d26be7cbbda4ddba6249e823a8b/types/ref/ref.go

package image

import "regexp"

var (
	hostPartS = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`
	portS     = `(?:` + regexp.QuoteMeta(`:`) + `[0-9]+)`
	ipv6PartS = `(?:[0-9a-fA-F]{1,4}:){0,7}[0-9a-fA-F]{1,4}`
	ipv6S     = `(?:` + regexp.QuoteMeta(`[`) + `(?:` +
		ipv6PartS + `|` + // uncompressed
		regexp.QuoteMeta(`::`) + ipv6PartS + `|` + // prefix compressed
		ipv6PartS + regexp.QuoteMeta(`::`) + ipv6PartS + `|` + // middle compressed
		ipv6PartS + regexp.QuoteMeta(`::`) + // suffix compressed
		`)` + regexp.QuoteMeta(`]`) + `)`
	localhostS  = `localhost`
	hostDomainS = `(?:` + hostPartS + `(?:(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)+` + regexp.QuoteMeta(`.`) + `?|` + regexp.QuoteMeta(`.`) + `))`
	hostUpperS  = `(?:[a-zA-Z0-9]*[A-Z][a-zA-Z0-9-]*[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[A-Z][a-zA-Z0-9]*)`
	registryS   = `(?:` +
		`(?:` + hostDomainS + `|` + hostUpperS + `|` + ipv6S + `|` + localhostS + `)` + portS + `?|` + // name with dotted domain, upper case, or IPv6 with optional port
		hostPartS + portS + // a short name with required port
		`)`

	registryRE = regexp.MustCompile(`^(` + registryS + `)$`)
)
