This is a functional proof of concept wrapper for `dep ensure -add` and `go get` which primarily detects the number of
third party dependencies for a package and asks for confirmation if this number exceeds 2 packages.

It will also print some additional information about the package provided by the go-search.org api.

The primary use-case being to prevent developers from unknowingly pulling in a simple package that comes with a ton of
baggage (dependencies of its own).

Usage:

```
go get github.com/Naatan/dep-areyousure
dep-areyousure github.com/spf13/cobra/cobra # dep ensure
dep-areyousure -get github.com/spf13/cobra/cobra # go get
```