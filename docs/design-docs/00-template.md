# Title

* Author(s): \<your name\>, \<your github alias\>
* Approver: \<kpt-maintainer\>

>    Every feature will need design sign off an PR approval from a core
>    maintainer.  If you have not got in touch with anyone yet, you can leave
>    this blank and we will try to line someone up for you.

## Why

Please provide a reason to have this feature.  For best results a feature should
be addressing a problem that is described in a github issue.  Link the issues
in this section.  The more user requests are linked here the more likely this
design is going to get prioritized on the roadmap.

It's good to include some background about the problem, but do not use that as a
substitute for real user feedback.

## Design

Please describe your solution. Please list any:

* new config changes
* interface changes
* design assumptions

For a new config change, please mention:

* Is it backwards compatible? If not, what is the migration and deprecation 
  plan?


## User Guide

This section should be written in the form of a detailed user guide describing 
the user journey. It should start from a reasonable initial state, often from 
scratch (Instead of starting midway through a convoluted scenario) in order 
to provide enough context for the reader and demonstrate possible workflows. 
This is a form of DDD (Documentation-Driven-Development), which is an effective 
technique to empathize with the user early in the process (As opposed to 
late-stage user-empathy sessions).

This section should be as detailed as possible. For example if proposing a CLI 
change, provide the exact commands the user needs to run, along with flag 
descriptions, and success and failure messages (Failure messages are an 
important part of a good UX). This level of detail serves two functions:

It forces the author and the readers to explicitly think about possible friction
points, pitfalls, failure scenarios, and corner cases (“A measure of a good 
engineer is not how clever they are, but whether they think about all the 
corner cases”). Makes it easier to author the user-facing docs as part of the 
development process (Ideally as part of the same PR) as opposed to it being an 
afterthought.

## Open Issues/Questions

Please list any open questions here in the following format:

### \<Question\>

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Alternatives Considered

If there is an industry precedent or alternative approaches please list them 
here as well as citing *why* you decided not to pursue those paths.

### \<Approach\>

Links and description of the approach, the pros and cons identified during the 
design.

# Review Process

Design doc creators should raise the pull request similar to [this](https://github.com/GoogleContainerTools/kpt/pull/2576).
Please name your documents with the design document number.  Pick n+1 from what's currently merged knowing that a slight
renaming might need to happen if multiple design docs get accepted simultaneously.
Please keep the name of the design doc in lower case with dashes in between words.

Reviewers are auto-assigned to review the PR. Optionally, doc owner can mention any
specific reviewers on the PR description. The turn around time for each review cycle
on the PR from kpt maintainers is 1-2 days. After maintainers add comments to the PR,
doc owner should respond to each of those comments and hit `Resolve conversation` button on the 
comment thread. Once all the comments are resolved, design doc creator should hit `Re-request review`
icon next to each reviewers' avatar in reviewers section. Reviewers will then be notified and, they will go through the
resolved comments and might reopen the comment threads, and the cycle continues.
Once all the reviewers approve the PR, it will be merged by the maintainers.
Any design doc PRs which are not attended by the doc creators for more than 2 weeks will be closed.
