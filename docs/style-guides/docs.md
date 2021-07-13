# kpt Docs Style Guide

1. CLI commands name should appear with inline quotes everywhere including in
   page titles:
   - Correct: `diff`
   - Incorrect: `Diff`
   - Incorrect: `The Diff tool`
2. Shell commands should be in `shell` code blocks and should contain exactly
   one command specifing using`$`:

   ```shell
   $ kpt <command>
   <output>
   ```

3. Links to other kpt doc pages should be absolute:
   - Correct: `[pkg]: /reference/cli/pkg/get/`
   - Incorrect: `[pkg]: ../../../reference/cli/pkg/get`
4. Flags must appear after positional args:

   - Correct:

   ```shell
   $ kpt fn eval my-package --image gcr.io/kpt-fn/search-replace
   ```

   - Incorrect:

   ```shell
   $ kpt fn eval --image gcr.io/kpt-fn/search-replace my-package
   ```

5. The name of the tool should always appear as small caps (even at start of
   sentences) and not in block quotes:
   - Correct: kpt
   - Incorrect: `kpt`
   - Incorrect: Kpt
   - Incorrect: KPT
6. References to a particular KRM group, version, kind, field should appear with
   inline quotes:
   - Correct: `ConfigMap`
   - Incorrect: ConfigMap
