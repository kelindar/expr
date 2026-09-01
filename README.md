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

- `Compile(source)` parses and optimizes an expression once.
- `Program.Eval(ctx, input, now)` evaluates a compiled program and returns a Go value. A zero `now` captures the current UTC time.
- `Program.JSON(input)` evaluates a program and returns validated JSON.
- `Program.Bool(input)` evaluates a program that must return a boolean.
- `Program.AppendJSON(dst, input)` evaluates a program and appends the JSON result to an existing buffer.
- `Program.Type()` returns the statically inferred JSON type, or an empty `Type` when the result is dynamic.
- `Evaluate(request)` compiles and evaluates one `Request` into a typed `Result`.
- `GetReference()` returns the machine-readable callable and validation-rule catalog.
- `Guide()` returns the embedded author guide.
- `Validate(raw)` checks a JSON result against size and structural output limits.

The root types are `Type`, `Program`, `Request`, `Result`, `Failure`,
`Reference`, and `Function`. `Program.Type()` returns `boolean`, `string`,
`integer`, `number`, `array`, or `object` when the compiler can prove the
result type. It returns an empty `Type` for dynamic results.

The helper packages expose the same operations as pure Go functions. Their
`Options()` functions register those operations in another Expr environment;
the root `Compile` function registers them automatically.
The expression names below map to the exported Go helpers, for example
`numeric.Sqrt` is available as `sqrt`. Every helper package exports
`Options()`. `encoding.CanonicalJSON` and `validate.Rules()` are Go-only
helpers.

## Expression language

Expressions support literals, arrays, maps, arithmetic, comparisons, Boolean
operators, ternaries, predicates, and pipes. The runtime adds these values and
helpers:

- `this` is the primary JSON input.
- `context` is optional caller-provided JSON metadata.
- `now()` returns the captured evaluation instant.
- `json(path)`, `raw(value)`, `time(value)`, and `duration(value)` handle JSON
  and time values.

The examples below are complete expressions. They assume the input is in
`this`; use `context` when a caller-provided value is needed.

### core

- `json(path)` reads a JSON path from `this`. Example: `json("items.#(state==\"done\")")`.
- `raw(value)` returns JSON text. Example: `raw(this.payload)`.
- `time(value)` converts seconds or nanoseconds to a time value. Example: `time(this.timestamp)`.
- `now()` returns the captured evaluation instant. Example: `now()`.
- `duration(value)` parses Go duration text. Example: `duration("5m")`.

### assignment

- `bucket(value, count, namespace?)` assigns a deterministic XXH3 bucket. Example: `bucket(this.user_id, 100, "checkout")`.

### collection

- `chunk(values, size)` splits an array into fixed-size chunks. Example: `chunk(this.items, 10)`.
- `zip(left, right)` combines equal-length arrays pair by pair. Example: `zip(this.keys, this.values)`.
- `merge(left, right)` shallow-merges maps, with values from `right` taking precedence. Example: `merge(this.defaults, this.options)`.
- `union(left, right)` returns a stable unique union. Example: `union(this.a, this.b)`.
- `intersection(left, right)` returns stable values present in both arrays. Example: `intersection(this.a, this.b)`.
- `difference(left, right)` returns stable values from `left` that are absent from `right`. Example: `difference(this.a, this.b)`.
- `lag(values, periods)` shifts values and inserts leading nulls. Example: `lag(this.values, 1)`.
- `cumsum(values)` returns cumulative numeric sums. Example: `cumsum(this.values)`.
- `diff(values)` returns adjacent numeric differences. Example: `diff(this.values)`.

### encoding

- `hash(value, algorithm)` hashes text or canonical JSON. Example: `hash(this.id, "sha256")`.
- `checksum(value, "crc32")` calculates a CRC32 IEEE checksum. Example: `checksum(this.payload, "crc32")`.
- `hexEncode(value)` encodes UTF-8 text as lowercase hexadecimal. Example: `hexEncode(this.id)`.
- `hexDecode(value)` decodes hexadecimal UTF-8 text. Example: `hexDecode("6869")`.
- `urlEncode(value, mode)` URL-encodes text in `query` or `path_segment` mode. Example: `urlEncode(this.query, "query")`.
- `urlDecode(value, mode)` URL-decodes text in `query` or `path_segment` mode. Example: `urlDecode(this.query, "query")`.

### network

- `normalizeIP(value)` normalizes IPv4 or IPv6 text. Example: `normalizeIP(this.ip)`.
- `inCIDR(ip, cidr)` tests IP membership without network access. Example: `inCIDR(this.ip, "10.0.0.0/8")`.
- `normalizeURL(value)` normalizes an absolute URL without reordering its query. Example: `normalizeURL(this.url)`.
- `normalizeHostname(value)` normalizes an IDNA hostname. Example: `normalizeHostname(this.host)`.

### numeric

- `sqrt(value)` returns the square root of a finite non-negative number. Example: `sqrt(9)`.
- `exp(value)` returns the natural exponential. Example: `exp(1)`.
- `clamp(value, min, max)` limits a number to ordered bounds. Example: `clamp(this.score, 0, 1)`.
- `roundTo(value, places)` rounds to a chosen number of decimal places. Example: `roundTo(1.234, 2)`.
- `log(value, base?)` returns a natural or base-specific logarithm. Example: `log(100, 10)`.
- `variance(values, method?)` calculates population or sample variance. Example: `variance(this.values)`.
- `stddev(values, method?)` calculates population or sample standard deviation. Example: `stddev(this.values, "sample")`.
- `quantile(values, p)` calculates a linearly interpolated quantile. Example: `quantile(this.values, 0.5)`.
- `covariance(x, y, method?)` calculates population or sample covariance. Example: `covariance(this.x, this.y)`.
- `correlation(x, y, method?)` calculates Pearson or Spearman correlation. Example: `correlation(this.x, this.y, "spearman")`.
- `dot(x, y)` calculates the dot product of equal-length numeric arrays. Example: `dot(this.a, this.b)`.
- `norm(values)` calculates the L2 norm. Example: `norm(this.vector)`.
- `normalize(values)` L2-normalizes a numeric array. Example: `normalize(this.vector)`.
- `distance(x, y, method?)` calculates numeric distance. Example: `distance(this.a, this.b, "manhattan")`.
- `similarity(x, y, method)` calculates cosine, Jaccard, Hamming, or Levenshtein similarity. Example: `similarity(this.a, this.b, "cosine")`.

### text

- `regexFind(value, pattern)` returns one RE2 match and its groups. Example: `regexFind(this.text, "(id-[0-9]+)")`.
- `regexFindAll(value, pattern)` returns all RE2 matches and their groups. Example: `regexFindAll(this.text, "id-[0-9]+")`.
- `regexReplace(value, pattern, replacement)` replaces matches with RE2 capture expansion. Example: `regexReplace(this.text, "foo", "bar")`.
- `normalizeUnicode(value, form?)` normalizes text as NFC, NFD, NFKC, or NFKD. Example: `normalizeUnicode(this.text)`.
- `removeDiacritics(value)` removes combining marks and returns NFC text. Example: `removeDiacritics(this.name)`.

### validate

- `is(value, rule, args...)` applies a named pure validation rule. Example: `is(this.email, "email")`.

The complete callable catalog is available from `GetReference().Functions`.
[`reference.md`](reference.md) contains the author guide, examples, result
types, validation rules, and language constraints. The upstream syntax is
documented in the [Expr language definition](https://expr-lang.org/docs/language-definition).

## Limits

Every entry point uses the same fixed limits:

- Expression source is limited to 8 KiB.
- The AST is limited to 1,000 nodes.
- JSON input and output are limited to 256 KiB each.
- Collections are limited to 10,000 entries.
- Nesting is limited to 64 levels.
- Expr VM memory is limited to 1,000,000 units.

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
