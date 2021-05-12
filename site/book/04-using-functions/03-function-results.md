In kpt, the counterpart to Unix philsophophy of "everything is a file" is "everything is a
Kubernetes resource". This also extends to the results of executing functions using `eval` or
`render`. In addition to providing a human-readable terminal output, these commands provide
structured results which can be consumed by other tools. This enables you to build robust UI layers
on top of kpt. For example:

- Create a custom dashboard that shows the results returned by functions
- Annotate a GitHub Pull Request with results returned by a validator function at the granularity of individuals fields

In both `render` and `eval`, structured results can be enabled using the `--results-dir` flag.
For example:

```shell
$ kpt fn render wordpress --results-dir /tmp
```

TODO
