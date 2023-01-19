Simple dependency scanner tool

Keeps license detection easy - relies on a static set of files.

To generate in a nice format:

```
go run . scan --binary ~/bin/kpt | \
  jq -r '.[] | [.name, (.licenseInfo.licenseURLs | join(" ")), .licenseInfo.license, "kpt", "YES", if .licenseInfo.mustShipCode then  "YES" else "NO" end ] | @csv'
```

Saving a copy in the repo is helpful to track changes over time:

```
go run . scan --binary ~/bin/kpt | jq . > results.txt
```

To generate a LICENSES text file (useful for embedding):

```
go run . scan --print --binary ~/bin/kpt | jq -r '.[] | ("================================================================================\n= " + .name + " =\n\n" + .licenseFiles[].contents + "\n\n")' > ../../licenses/kpt.txt
```