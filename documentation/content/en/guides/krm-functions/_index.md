---
title: KRM Function Developer Guide
linkTitle: KRM Function Developer Guide
description: Write your own KRM functions with the Go SDK.
toc_hide: false
menu:
  main:
    parent: "Guides"
---
This guide walks through writing KRM functions with the
[Go SDK](https://github.com/kptdev/krm-functions-sdk). Start with the tutorial,
then dig into the topic guides as needed.

- [Tutorial]({{% relref "/guides/krm-functions/tutorial" %}}) — build a working
  function end to end, with embedded documentation, golden tests, and support for
  `--help`, `--doc`, and standalone file mode.
- [Interfaces]({{% relref "/guides/krm-functions/interfaces" %}}) — choose between
  `fn.Runner` (transformers, validators) and `fn.ResourceListProcessor`
  (generators, complex functions).
- [Testing]({{% relref "/guides/krm-functions/testing" %}}) — golden test patterns
  and unit testing in depth.
- [Containerizing]({{% relref "/guides/krm-functions/containerizing" %}}) — package
  your function as a container image.

For a complete working example, see
[`go/get-started/`](https://github.com/kptdev/krm-functions-sdk/tree/main/go/get-started).
