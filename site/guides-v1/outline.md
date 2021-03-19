# outline

1. Getting started
    1. Installation
    1. Quick start: Jump into an example quickly showing all the functionality (pkg, fn, live) in an e2e workflow.
1. Concepts: An introduction to fundamental concepts in kpt.
    1. Packages
        1. Package lifecycle/workflow
    1. Functions
1. Packages
    1. Fetching an existing package
    1. Exploring a fetched package
    1. Customizing a package
        1. Manual editing
        1. Using functions to automate customization and validation (Link to next chapter)
    1. Rebasing a package
    1. Creating a new package
    1. Publishing a package to Git
    1. Working with subpackages
1. Functions
    1. Running functions imperatively
        1. Example: Set namespaces on all resources
        1. Example: A smarter search and replace
        1. Scripting with functions
    1. Declaring and running functions in a package
        1. Example: Set namespaces on all resources
        1. Example: Enforcing policies
    1. Developing functions
        1. Developing custom function images
            1. Functions specification
            1. Developing in Golang
            1. Developing in Typescript
        1. Executable configuration
            1. Starlark function
            1. Gatekeeper function
1. Apply
    1. Apply a package to the cluster
    1. Lifecycle of applied resources