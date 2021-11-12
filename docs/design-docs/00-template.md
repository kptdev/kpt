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