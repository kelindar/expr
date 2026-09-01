<p align="center">
    <img width="225" height="100" src=".github/logo.png" border="0" alt="kelindar/expr">
    <br>
    <img src="https://img.shields.io/github/go-mod/go-version/kelindar/expr" alt="Go Version">
    <a href="https://pkg.go.dev/github.com/kelindar/expr"><img src="https://pkg.go.dev/badge/github.com/kelindar/expr" alt="PkgGoDev"></a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
    <a href="https://coveralls.io/github/kelindar/expr"><img src="https://coveralls.io/repos/github/kelindar/expr/badge.svg" alt="Coverage"></a>
</p>

# Bounded JSON Expressions for Go

This package is a small, deterministic expression evaluator for JSON payloads. It
is built for services that need user-authored logic over structured data without
opening the door to arbitrary code, storage access, or network I/O.

Expressions read the input as `this`, optional caller metadata as `context`, and
a single captured evaluation time. Compilation and evaluation enforce fixed
source, input, output, nesting, and collection limits. The same contract is used
for direct Go calls and HTTP-style evaluation envelopes.

It fits naturally alongside other Kelindar data libraries:

- [`kelindar/storage`](https://github.com/kelindar/storage) stores typed JSON
  resources and exposes query filters over stored fields.
- [`kelindar/column`](https://github.com/kelindar/column) keeps columnar,
  in-memory collections with bitmap indexes for fast scans.
- [`kelindar/tile`](https://github.com/kelindar/tile) provides a cache-friendly
  2D grid for spatial workloads.

`expr` does not read from those stores directly. Instead, callers pass JSON input
into the evaluator and use the result to filter, transform, score, or route
resources and events.

- **Bounded.** Fixed limits on source size, AST size, JSON input/output, nesting,
  collection entries, and VM memory.
- **Deterministic.** No randomness, wall-clock timeouts, files, network, or
  arbitrary code execution.
- **Typed.** Results use a stable envelope for booleans, numbers, strings,
  arrays, objects, time, duration, and structured failures.
- **Fast paths.** Compile-time JSON projection and boolean predicates avoid the
  full VM when the expression shape allows it.
- **Documented surface.** `Functions()`, `Guide()`, and validation rules describe
  the callable catalog exposed to authors.

# Installation

```sh
go get github.com/kelindar/expr
```

# Quick start

Compile once, evaluate many times against JSON input:

```go
package main

import (
	"fmt"

	"github.com/kelindar/expr"
)

func main() {
	program, err := expr.Compile(`this.price * 1.2`)
	if err != nil {
		panic(err)
	}

	value, err := expr.Eval(program, []byte(`{"price": 10}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(value) // 12
}
```

For HTTP handlers and tools that need the shared request/response contract:

```go
result := expr.Evaluate(expr.Request{
	Expression: `this.enabled && this.score > 0.8`,
	Input:      []byte(`{"enabled": true, "score": 0.91}`),
})
```

Successful results use `result.Type` and `result.Out`. Expression failures stay
inside the envelope with `type: "error"` and a stable failure code such as
`invalid_input`, `compile_error`, `evaluation_error`, or `limit_exceeded`.

# Evaluation API

The root package exposes a small surface:

- `Compile` parses and optimizes an expression.
- `Eval`, `EvalWithContext`, `JSON`, `Bool`, and `AppendJSONUnchecked` evaluate a
  compiled program.
- `Evaluate` compiles and evaluates one `Request` and returns the typed `Result`
  envelope.
- `Functions`, `Guide`, and `Reference` expose the documented callable catalog.
- `ValidateOutput` checks a JSON result against the output contract.

`Program.JSONType()` reports a statically known JSON Schema result type when the
compiler can infer one.

## Environment

Authors can use:

- `this` for the primary JSON input.
- `context` for separate caller-provided JSON metadata.
- `now()` for the evaluation instant, captured once per evaluation.
- `json(path)`, `raw(value)`, `time(value)`, and `duration(value)` for JSON and
  time helpers documented in `reference.md`.

Pipes, predicates, arrays, maps, literals, and the built-in domains below compose
into larger expressions. See `reference.md` for the author-facing language guide.

## Safety limits

Every entry point shares the same limits:

| Limit | Value |
| --- | --- |
| Expression source | 8 KiB |
| AST nodes | 1,000 |
| JSON input and output | 256 KiB each |
| Collection entries | 10,000 |
| Nesting depth | 64 |
| Expr VM memory | 1,000,000 |

These limits are not caller-configurable.

# Built-in domains

Custom callables are grouped into focused packages and registered automatically
during compilation:

| Package | Examples |
| --- | --- |
| `assignment` | deterministic `bucket` |
| `collection` | `chunk`, `zip`, `merge`, `union`, `intersection`, `lag`, `diff` |
| `encoding` | `hash`, `checksum`, `hexEncode`, `urlEncode`, canonical JSON |
| `network` | `normalizeIP`, `inCIDR`, `normalizeHostname`, `normalizeURL` |
| `numeric` | `sqrt`, `clamp`, `quantile`, `correlation`, `distance`, `similarity` |
| `text` | `regexFind`, `regexReplace`, `normalizeUnicode`, `removeDiacritics` |
| `validate` | `is`, `matches`, `oneOf`, `allOf`, and related predicates |

`validate` also publishes the rule catalog returned by `Reference()`.

# Package layout

```text
/                     root evaluator, contract, catalog, limits
/assignment           deterministic assignment helpers
/collection           collection transforms and set operations
/encoding             hashing, checksums, and canonical JSON
/network                IP, CIDR, URL, and hostname normalization
/numeric                statistics, vectors, and similarity helpers
/text                   Unicode and regular-expression helpers
/validate               validation predicates and rule metadata
/bench                  standalone fixture benchmark module
/internal/fixture       shared YAML fixture loader for tests and benches
/internal/jcs           vendored RFC 8785 JSON canonicalization (Apache 2.0)
```

`internal/*` packages are not part of the public API.

# Benchmarks

Fixture benchmarks live in a separate module:

```sh
(cd bench && go run .)
```

# Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

# Contributing

We are open to contributions. Please keep changes focused and run the relevant
tests before sending a pull request. This library is maintained by
[Roman Atachiants](https://www.linkedin.com/in/atachiants/).

# License

Expr is licensed under the MIT License.
