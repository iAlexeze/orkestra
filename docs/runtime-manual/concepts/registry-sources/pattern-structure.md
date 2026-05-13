# The Pattern Structure

Before loading any source file, Orkestra validates that the pulled pattern
contains all five required files, each non-empty:

```
crd.yaml          the CRD definition to install in the cluster
katalog.yaml      operator behavior — reconcile templates, conversion rules
komposer.yaml     an example showing how to import and override this pattern
cr.yaml           an example custom resource to test with
README.md         documentation — fields, overrides, and examples
```

!!! warning "Fail fast"
    If any of the five files is missing or empty, Orkestra refuses to load
    the pattern and prints a clear error listing every violation. This check
    runs during `ork validate` — before any reconciliation begins.

    See [Error reference](./error-reference.md/#structure-validation-errors).

This is intentional. A pattern without documentation is not ready for
distribution. A pattern without an example CR cannot be tested without
reading the source. The five-file requirement is the minimum bar for a
pattern to be trustworthy.

!!! tip "Building your own pattern"
    If you are publishing your own registry pattern, run
    `ork validate --file katalog.yaml` inside the pattern directory before
    pushing. Orkestra validates structure as part of that command and will
    surface missing or empty files before they reach consumers.

