## Scenario:

When using template solutions like Helm I am able to provide conditional statements based on parameter values.  This allows me to ask the user for a little bit of information and generate a lot of boilerplate configuration.

## Problems:

1. Over time the templating logic becomes its own language that becomes very complex.  Debugging the template generation becomes a task of its own.
1. The interplay between different conditionals and loops is interleaved in the template making it hard to understand what exactly is configuration and what is the logic that alters the configuration.
1. Templates are generally monolithic, when a change is introduced the package consumers need to either pay the cost of updating or the new consumers pay the cost of having to decipher more optional parameters.

## Solutions:

1. When a complex configuration needs to be generated the package author can create a generator function using turing complete languages and debugging tools.  Example of such a function is [folder generation].  The output of the function is plain old KRM.

[folder generation]: https://catalog.kpt.dev/generate-folders/v0.1/