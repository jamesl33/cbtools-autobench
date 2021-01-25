cbtools-autobench
-----------------

[![Go Reference](https://pkg.go.dev/badge/github.com/jamesl33/cbtools-autobench.svg)](https://pkg.go.dev/github.com/jamesl33/cbtools-autobench)

An automatic benchmarking tools designed to benchmark Couchbase tools, written with the intention of producing reliable
benchmarks and to reduce the feedback loop for changes made to performance critical components.

Contributing
------------

To contribute to this repository please feel free to create pull requests on GitHub using a fork of this repository.
Make sure you have configured the git hooks so that the code is linted and formatted before uploading the patch.

For the git hooks the following dependencies are required:

```
gofmt
gofumpt
goimports
golangci-lint
```

Once you have installed the dependencies set the git hooks path by using the command below:

```
git config core.hooksPath .githooks
```

## Coding style

In this section we will cover notes on the exact coding style to use for this codebase. Most of the style rules are
enforced by the linters, so here we will only cover ones that are not.

### Documenting

- All exported functions should have a matching docstring.
- Any non-trivial unexported function should also have a matching docstring. Note this is left up to the developer and
  reviewer consideration.
- Docstrings must end on a full stop (`.`).
- Comments must be wrapped at 120 characters.
- Notes on interesting/unexpected behavior should have a newline before them and use the `// NOTE:` prefix.

License
-------
Copyright 2021 Couchbase Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
