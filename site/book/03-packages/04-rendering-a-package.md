Regardless of how you have edited the package, you want to _render_ the package:

```shell
$ kpt fn render wordpress
```

?> Refer to the [render command reference][render-doc] for usage.

`render` is a critical step in the package lifecycle. At a high level, it
perform the following steps:

1. Enforces package preconditions. For example, it validates the `Kptfile`.
2. Executes functions declared in the package hierarchy in a depth-first order.
   By default, the packages are modified in-place.
3. Guarantees package postconditions. For example, it enforces a consistent
   formatting of resources, even though a function (developed by different
   people using different toolchains) may have modified the formatting in some
   way.

[Chapter 4] discusses different ways of running functions in detail.

[render-doc]: /reference/cli/fn/render/
[chapter 4]: /book/04-using-functions/
