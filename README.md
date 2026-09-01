<p align="center">
    <img width="225" height="100" src=".github/logo.png" border="0" alt="kelindar/expr">
    <br>
    <img src="https://img.shields.io/github/go-mod/go-version/kelindar/expr" alt="Go Version">
    <a href="https://pkg.go.dev/github.com/kelindar/expr"><img src="https://pkg.go.dev/badge/github.com/kelindar/expr" alt="PkgGoDev"></a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
    <a href="https://coveralls.io/github/kelindar/expr"><img src="https://coveralls.io/repos/github/kelindar/expr/badge.svg" alt="Coverage"></a>
</p>

Copyright (c) Roman Atachiants and contributors. All rights reserved.
Licensed under the MIT license. See [LICENSE](LICENSE).

# Bounded JSON Expressions for Go

This package is a small, deterministic expression evaluator for JSON payloads. It
is built for services that need user-authored logic over structured data without
opening the door to arbitrary code, storage access, or network I/O.

This library extends [Expr](https://github.com/expr-lang/expr); it is not a
standalone expression language. It uses Expr's parser, compiler, VM, and language
definition, then adds bounded JSON evaluation, deterministic helper packages,
fixed-time evaluation, and a typed request/result contract. Expr is Copyright
(c) 2018 Anton Medvedev and is also distributed under the MIT License. See the
[upstream license](https://github.com/expr-lang/expr/blob/master/LICENSE) and
[language definition](https://expr-lang.org/docs/language-definition).

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
- **Documented API.** `GetReference()` and `Guide()` describe the callable
  catalog exposed to authors.

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
	"time"

	"github.com/kelindar/expr"
)

func main() {
	program, err := expr.Compile(`this.price * 1.2`)
	if err != nil {
		panic(err)
	}

	value, err := program.Eval(nil, []byte(`{"price": 10}`), time.Time{})
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

# Public Go API

The root package keeps the runtime surface small:

| Symbol | Purpose |
| --- | --- |
| `Compile(source)` | Parse and optimize an expression once. |
| `Program.Eval(ctx, input, now)` | Evaluate a compiled program and return a Go value. A zero `now` captures the current UTC time. |
| `Program.JSON(input)` | Evaluate and return a validated JSON result. |
| `Program.Bool(input)` | Evaluate a program that must return a boolean. |
| `Program.AppendJSON(dst, input)` | Evaluate and append the JSON result to an existing buffer. |
| `Program.Type()` | Return the statically inferred JSON type, or an empty `Type` when dynamic. |
| `Evaluate(request)` | Compile and evaluate one `Request` into the typed `Result` envelope. |
| `GetReference()` | Return the machine-readable callable and validation-rule catalog. |
| `Guide()` | Return the embedded author-facing language guide. |
| `Validate(raw)` | Check a JSON result against size and structural output limits. |

The root types are `Type`, `Program`, `Request`, `Result`, `Failure`,
`Reference`, and `Function`. `Program.Type()` returns one of `boolean`,
`string`, `integer`, `number`, `array`, or `object` when the compiler can prove
the result type. It returns an empty `Type` for dynamic results.

The helper packages also expose their pure Go functions. Their `Options()`
functions are used to register the same callables in another Expr environment;
the root `Compile` function registers them automatically.

| Package | Exported functions |
| --- | --- |
| `assignment` | `Bucket`, `Options` |
| `collection` | `Chunk`, `Cumsum`, `Diff`, `Difference`, `Intersection`, `Lag`, `Merge`, `Options`, `Union`, `Zip` |
| `encoding` | `CanonicalJSON`, `Checksum`, `Hash`, `HexDecode`, `HexEncode`, `Options`, `URLDecode`, `URLEncode` |
| `network` | `InCIDR`, `NormalizeHostname`, `NormalizeIP`, `NormalizeURL`, `Options` |
| `numeric` | `Clamp`, `Correlation`, `Covariance`, `Distance`, `Dot`, `Exp`, `Log`, `Norm`, `Normalize`, `Options`, `Quantile`, `RoundTo`, `Similarity`, `Sqrt`, `StdDev`, `Variance` |
| `text` | `NormalizeUnicode`, `Options`, `RegexFind`, `RegexFindAll`, `RegexReplace`, `RemoveDiacritics` |
| `validate` | `Is`, `Options`, `Rules` |

The complete expression callable catalog is returned by
`GetReference().Functions`. The author guide, examples, result types, safety
limits, and language rules live in [`reference.md`](reference.md). The
catalog is also available at runtime, so clients can render the exact version
of the reference they are using.

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

`validate` also publishes the rule catalog returned by `GetReference()`.

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

This extension is licensed under the MIT License. See [LICENSE](LICENSE).

It extends [Expr](https://github.com/expr-lang/expr), Copyright (c) 2018 Anton
Medvedev, also under the MIT License. The vendored
[`internal/jcs`](internal/jcs) package retains its upstream Apache 2.0 notice;
see [`internal/jcs/LICENSE`](internal/jcs/LICENSE).
