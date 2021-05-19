# kpt Error Messages Style Guide

Validation error messages and documentation are an important part of the UX in kpt. In fact, validation error messages are most likely the first thing users experience. The general philosophy here is to have **precise**, **actionable**, and **consistent** error messages. This document is based on the similar style guide for Config Sync (go/nomos-style).

## Errors vs. Documentation

In most cases, the error message should provide enough information to resolve the issue. Prefer short descriptions that explain what caused the specific error and what to do to fix it. More extensive explanations and general rules (if they take more than a sentence to describe) are for documentation (which is automatically linked in error messages).

Generally, documentation should expand on what the error message says and help the reader understand how to prevent future mistakes.

## Error Message Rules

In the table below, origin and scope of rules are denoted with a prefix

- **K** is inherited from [Kubernetes conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#validation); rules with prefix
- **R** are specific to Kpt.

<table>
  <tr>
   <td>
<strong>Rule</strong>
   </td>
   <td><strong>Description</strong>
   </td>
   <td><strong>Examples</strong>
   </td>
  </tr>
  <tr>
   <td><strong>K1</strong>
   </td>
   <td>Where possible, tell users what they CAN instead of what they CANNOT do.
   </td>
   <td>
   </td>
  </tr>
  <tr>
   <td><strong>K2</strong>
   </td>
   <td>When asserting a requirement in the positive, use "must". Words like "should" imply that the assertion is optional, and must be avoided.
   </td>
   <td><em>"must be greater than 0"</em>
<p>
<em>"must match regex '[a-z]+'"</em>
   </td>
  </tr>
  <tr>
   <td><strong>K3</strong>
   </td>
   <td>When asserting a formatting requirement in the negative, use "must not". Words like "should not" imply that the assertion is optional, and must be avoided.
   </td>
   <td><em>"must not contain '..'"</em>
   </td>
  </tr>
  <tr>
   <td><strong>K5</strong>
   </td>
   <td>When referencing a user-provided string value, indicate the literal in quotes.
<p>
<strong>N</strong> Use quotes (%q format specifier in Go) around user-provided values. This includes file paths.
   </td>
   <td><em>"must not contain '..'"</em>
   </td>
  </tr>
  <tr>
   <td><strong>K6</strong>
   </td>
   <td>When referencing a field name, indicate the name in back-quotes.
<p>
<strong>N</strong> Where unclear from the message, reference the full field path.
   </td>
   <td><em>"must be greater than <code>`spec.request.size`</code>"</em>
   </td>
  </tr>
  <tr>
   <td><strong>K7</strong>
   </td>
   <td>When specifying inequalities, use words rather than symbols. Do not use words like "larger than", "bigger than", "more than", "higher than", etc.
   </td>
   <td><em>"must be less than 256"</em>
<p>
<em>"must be greater than or equal to 0". </em>
   </td>
  </tr>
  <tr>
   <td><strong>K8</strong>
   </td>
   <td>When specifying numeric ranges, use inclusive ranges when possible.
   </td>
   <td>
   </td>
  </tr>
  <tr>
   <td><strong>R1</strong>
   </td>
   <td>If wrapping a runtime error, such as the result of a failed API Server call, use <code>%w formatting verb in fmt.Errorf()(err)</code> to include the root cause error. Refer to Go lang error enhancements in Go 1.13 (https://blog.golang.org/go1.13-errors)<code>.</code>
   </td>
   <td>
   </td>
  </tr>
</table>
