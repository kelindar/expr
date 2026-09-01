<!-- Copyright (c) Roman Atachiants and contributors. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root. -->

# Expression reference

An expression is a small, pure calculation. It reads the input as `this` and
optional caller data as `context`, then combines values with literals,
operators, pipes, predicates, and the documented callables in the Reference
catalog. It cannot access storage, secrets, files, the network, or arbitrary
code.

## Start with the input

The test-data editor supplies the JSON value exposed as `this`:

```expr
this.customer.email
```

Use dotted paths for known fields, brackets for a dynamic key, and ordinary
JSON literals for values you want to compare or construct:

```expr
this.items[0].price * 1.05
this["status"] == "ready"
{name: this.customer.name, active: this.enabled == true}
```

`context` is separate from the input and is useful for caller-provided
metadata. `this` and `context` are read-only. Missing values behave like
`nil`; use comparisons and conditional expressions to handle them explicitly.

## Compose expressions

Use arithmetic (`+`, `-`, `*`, `/`, `%`), comparisons (`==`, `!=`, `<`, `<=`,
`>`, `>=`), Boolean operators (`&&`, `||`, `!`), ternaries (`condition ? a :
b`), arrays, maps, and pipes. A pipe passes the value on its left as the first
argument to the callable on its right:

```expr
this.orders
  | filter({.state == "paid"})
  | map({.total})
  | sum()
```

Predicates and transforms use `{...}` with `.` referring to the current item.
For example, `filter(this.items, {.enabled})` keeps enabled items and
`map(this.items, {.name})` projects their names.

## Call a function

The catalog beside this guide is the source of truth for available callables.
Each entry explains what the function does, its argument shape, the value it
returns, constraints or caveats, and a copyable example. Search by name,
domain, argument, or behavior when you know the outcome but not the function.

Examples:

```expr
roundTo(this.price * 1.05, 2)
normalizeUnicode(this.name, "nfkc")
is(this.email, "email") && inCIDR(this.ip, "10.0.0.0/8")
bucket(this.user_id, 100, "checkout")
```

The core helpers are `json(path)` for a JSON path from `this`,
`raw(value)` for JSON text, `time(value)` for Unix seconds or nanoseconds,
`duration(value)` for Go duration text, and `now()` for the evaluation instant.
`now()` is captured once when evaluation starts, so repeated calls in one
evaluation are identical.

## Results and failures

The evaluator returns a typed envelope. Successful values are `null`,
`boolean`, `integer`, `number`, `string`, `array`, `object`, `time`, or
`duration`. Time is rendered as RFC 3339 text and duration as Go duration text.
An expression mistake is still an HTTP-successful response with `type: "error"`;
its output contains a stable code (`invalid_input`, `compile_error`,
`evaluation_error`, or `limit_exceeded`) and a useful message. Compile and
runtime locations are one-based when the language provides them.

## Fixed safety limits

Every entry point uses the same limits: source up to 8 KiB, AST up to 1,000
nodes, input and output JSON up to 256 KiB, at most 10,000 collection entries,
nesting up to 64, and Expr VM memory up to 1,000,000. These limits are not
caller-configurable. Expressions are synchronous and pure; there is no
wall-clock timeout, randomness, I/O, JavaScript, or code node.

## Validation rules

`is(value, "rule_name", args...)` applies one named pure predicate. Rule names
and their accepted arguments are listed in the catalog response. A wrong value
type returns `false`; an unknown rule or invalid rule arguments returns an
`evaluation_error`. The rules do not query storage or external services.

## Language provenance

The expression syntax and built-in behavior are adapted from Expr's language
definition. This guide and catalog describe the available environment, limits,
result contract, and added functions. See the upstream documentation
at https://expr-lang.org/docs/language-definition and retain Expr's MIT
attribution when redistributing the language integration.
