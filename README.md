<p align="center">
    <img width="225" height="100" src=".github/logo.png" border="0" alt="kelindar/expr">
    <br>
    <img src="https://img.shields.io/github/go-mod/go-version/kelindar/expr" alt="Go Version">
    <a href="https://pkg.go.dev/github.com/kelindar/expr"><img src="https://pkg.go.dev/badge/github.com/kelindar/expr" alt="PkgGoDev"></a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
</p>

# Bounded JSON expressions for Go

`kelindar/expr` extends [Expr](https://github.com/expr-lang/expr) with a
bounded runtime for evaluating expressions over JSON. It uses Expr's parser,
compiler, and VM, then adds deterministic JSON values, helper packages, fixed
evaluation time, output limits, and a typed request/result contract.

Compile a program once and evaluate it as many times as needed. Expressions can
read the input as `this` and optional caller metadata as `context`, but cannot
access files, secrets, storage, networks, or arbitrary Go code.

- **Bounded.** Source, AST, JSON, collection, nesting, and VM memory limits are
  enforced at every entry point.
- **Deterministic.** Evaluation is synchronous and pure. `now()` is captured
  once for each evaluation.
- **JSON-first.** Results can be returned as Go values, JSON, booleans, or a
  typed result envelope.
- **Fast.** Compile-time JSON projections and boolean predicates skip the VM
  when the expression shape allows it.

## Installation

Requires Go 1.27 or newer:

```sh
go get github.com/kelindar/expr
```

## Quick start

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

For HTTP handlers and tools, `Evaluate` returns the shared typed envelope:

```go
result := expr.Evaluate(expr.Request{
	Expression: `this.enabled && this.score > 0.8`,
	Input:      []byte(`{"enabled": true, "score": 0.91}`),
})
```

Successful results expose `result.Type` and `result.Out`. Failures stay in the
envelope with a stable type and code such as `invalid_input`, `compile_error`,
`evaluation_error`, or `limit_exceeded`.

## Public Go API

The root package contains the evaluator and the request/result contract:

| Symbol | Purpose |
| --- | --- |
| `Compile(source)` | Parse and optimize an expression once. |
| `Program.Eval(ctx, input, now)` | Evaluate a compiled program and return a Go value. A zero `now` captures the current UTC time. |
| `Program.JSON(input)` | Evaluate and return a validated JSON result. |
| `Program.Bool(input)` | Evaluate a program that must return a boolean. |
| `Program.AppendJSON(dst, input)` | Evaluate and append the JSON result to an existing buffer. |
| `Program.Type()` | Return the statically inferred JSON type, or an empty `Type` when dynamic. |
| `Evaluate(request)` | Compile and evaluate one `Request` into a typed `Result`. |
| `GetReference()` | Return the machine-readable callable and validation-rule catalog. |
| `Guide()` | Return the embedded author guide. |
| `Validate(raw)` | Check a JSON result against size and structural output limits. |

The root types are `Type`, `Program`, `Request`, `Result`, `Failure`,
`Reference`, and `Function`. `Program.Type()` returns `boolean`, `string`,
`integer`, `number`, `array`, or `object` when the compiler can prove the
result type. It returns an empty `Type` for dynamic results.

The helper packages expose the same operations as pure Go functions. Their
`Options()` functions register those operations in another Expr environment;
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

## Expression language

Expressions support literals, arrays, maps, arithmetic, comparisons, Boolean
operators, ternaries, predicates, and pipes. The runtime adds these values and
helpers:

- `this` is the primary JSON input.
- `context` is optional caller-provided JSON metadata.
- `now()` returns the captured evaluation instant.
- `json(path)`, `raw(value)`, `time(value)`, and `duration(value)` handle JSON
  and time values.

The complete callable catalog is available from `GetReference().Functions`.
[`reference.md`](reference.md) contains the author guide, examples, result
types, validation rules, and language constraints. The upstream syntax is
documented in the [Expr language definition](https://expr-lang.org/docs/language-definition).

## Limits

Every entry point uses the same fixed limits:

| Limit | Value |
| --- | --- |
| Expression source | 8 KiB |
| AST nodes | 1,000 |
| JSON input and output | 256 KiB each |
| Collection entries | 10,000 |
| Nesting depth | 64 |
| Expr VM memory | 1,000,000 |

These limits are not caller-configurable.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

Run the fixture benchmarks with:

```sh
cd bench && go run .
```

## License

Copyright (c) Roman Atachiants and contributors. All rights reserved.

This extension is licensed under the [MIT License](LICENSE). It extends
[Expr](https://github.com/expr-lang/expr), Copyright (c) 2018 Anton Medvedev,
which is also MIT-licensed. The vendored [`internal/jcs`](internal/jcs)
package retains its Apache 2.0 notice in [`internal/jcs/LICENSE`](internal/jcs/LICENSE).
