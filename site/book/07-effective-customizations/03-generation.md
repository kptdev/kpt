## Scenario:

When using template languages I am able to provide conditional statements based 
on parameter values.  This allows me to ask the user for a little bit of 
information and generate a lot of boilerplate configuration.  Some template 
languages like [Jinja] are very robust and feature rich.

## Problems:

1. Increased usage and additional edge cases make a template a piece of code 
that requires testing and debugging.
1. The interplay between different conditionals and loops is interleaved in the template making it hard to understand what exactly is configuration and what is 
the logic that alters the configuration.  The consumer is left with one choice 
supply different parameter values, execute the template rendering code and see 
what happens.
1. Templates are generally monolithic, when a change is introduced the package consumers need to either pay the cost of updating or the new consumers pay the 
cost of having to decipher more optional parameters.

## Solutions:

1. When the generated configuration is simple consider just using a sub-package
and running customizations using [single value replacement] techniques.
1. When a complex configuration needs to be generated the package author can 
create a generator function using turing complete languages and debugging tools.  Example of such a function is [folder generation].  The output of the function 
is plain old KRM.

[folder generation]: https://catalog.kpt.dev/generate-folders/v0.1/
[Jinja]: https://palletsprojects.com/p/jinja/
[single value replacement]: /book/07-effective-customizations/01-single-value-replacement.md