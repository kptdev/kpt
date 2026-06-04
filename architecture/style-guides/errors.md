# kpt Error Messages Style Guide

Validation error messages and documentation are an important part of the UX in
kpt. In fact, validation error messages are most likely the first thing users
experience. The general philosophy here is to have **precise**, **actionable**,
and **consistent** error messages.

## Errors vs. Documentation

In most cases, the error message should provide enough information to resolve
the issue. Prefer short descriptions that explain what caused the specific error
and what to do to fix it. More extensive explanations and general rules (if they
take more than a sentence to describe) are for documentation (which is
automatically linked in error messages).

Generally, documentation should expand on what the error message says and help
the reader understand how to prevent future mistakes.

## Error Message Rules

In the table below, origin and scope of rules are denoted with a prefix

- **K** is inherited from
  [Kubernetes conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#validation);
  rules with prefix
- **R** are specific to Kpt.

<!-- prettier-ignore -->
|  Rule  | Description  | Examples  |
| --- | --- | --- |
| K1  | Where possible, tell users what they CAN instead of what they CANNOT do.  |   |
| K2  | When asserting a requirement in the positive, use "must". Avoid words like "should" as they imply that the assertion is optional.  | a) "must be greater than 0"  b) "must match regex '[a-z]+'"  |
| K3  | When asserting a formatting requirement in the negative, use "must not". Avoid words like "should not" as they imply that the assertion is optional.  | "must not contain '..'"  |
| K5  | When referencing a user-provided string value, indicate the literal in quotes.  N Use quotes (%q format specifier in Go) around user-provided values. This includes file paths.  | "must not contain '..'"  |
| K6  | When referencing a field name, indicate the name in back-quotes.  Where it's unclear from the message, reference the full field path.  | "must be greater than `spec.request.size`"  |
| K7  | When specifying inequalities, use words rather than symbols. Do not use words like "larger than", "bigger than", "more than", "higher than", etc.  | a) "must be less than 256"  b) "must be greater than or equal to 0".   |
| K8  | When specifying numeric ranges, use inclusive ranges when possible.  |   |
| R1  | If wrapping a runtime error, such as the result of a failed API Server call, use %w formatting verb in fmt.Errorf()(err) to include the root cause error. Refer to [Go error enhancements in 1.13](https://blog.golang.org/go1.13-errors).  |   |
