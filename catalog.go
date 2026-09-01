package expr

import (
	"embed"
	"sort"

	"github.com/expr-lang/expr/builtin"
	"github.com/kelindar/expr/validate"
)

//go:embed reference.md
var referenceFS embed.FS

// Function describes one documented expression callable.
type Function struct {
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Returns     string `json:"returns"`
	Notes       string `json:"notes,omitempty"`
	Insert      string `json:"insert"`
	Example     string `json:"example"`
}

type functionDoc struct {
	signature   string
	summary     string
	description string
	usage       string
	returns     string
	notes       string
}

// Reference is the authenticated expression language reference response.
type Reference struct {
	Version   string          `json:"version"`
	Upstream  string          `json:"upstream"`
	Guide     string          `json:"guide"`
	Functions []Function      `json:"functions"`
	Rules     []validate.Rule `json:"rules"`
}

var customFunctions = []Function{
	{Domain: "numeric", Name: "sqrt", Signature: "sqrt(value)", Summary: "Square root of a finite non-negative number.", Insert: "sqrt()", Example: "sqrt(9)"},
	{Domain: "numeric", Name: "exp", Signature: "exp(value)", Summary: "Natural exponential.", Insert: "exp()", Example: "exp(1)"},
	{Domain: "numeric", Name: "clamp", Signature: "clamp(value, min, max)", Summary: "Clamp a number to ordered bounds.", Insert: "clamp()", Example: "clamp(this.score, 0, 1)"},
	{Domain: "numeric", Name: "roundTo", Signature: "roundTo(value, places)", Summary: "Round to -15 through 15 decimal places.", Insert: "roundTo()", Example: "roundTo(1.234, 2)"},
	{Domain: "numeric", Name: "log", Signature: "log(value, base?)", Summary: "Natural or base-specific logarithm.", Insert: "log()", Example: "log(100, 10)"},
	{Domain: "numeric", Name: "variance", Signature: "variance(values, method?)", Summary: "Population or sample variance.", Insert: "variance()", Example: "variance(this.values)"},
	{Domain: "numeric", Name: "stddev", Signature: "stddev(values, method?)", Summary: "Population or sample standard deviation.", Insert: "stddev()", Example: "stddev(this.values, \"sample\")"},
	{Domain: "numeric", Name: "quantile", Signature: "quantile(values, p)", Summary: "Linearly interpolated quantile.", Insert: "quantile()", Example: "quantile(this.values, 0.5)"},
	{Domain: "numeric", Name: "covariance", Signature: "covariance(x, y, method?)", Summary: "Population or sample covariance.", Insert: "covariance()", Example: "covariance(this.x, this.y)"},
	{Domain: "numeric", Name: "correlation", Signature: "correlation(x, y, method?)", Summary: "Pearson or Spearman correlation.", Insert: "correlation()", Example: "correlation(this.x, this.y, \"spearman\")"},
	{Domain: "numeric", Name: "dot", Signature: "dot(x, y)", Summary: "Dot product of equal-length numeric arrays.", Insert: "dot()", Example: "dot(this.a, this.b)"},
	{Domain: "numeric", Name: "norm", Signature: "norm(values)", Summary: "L2 norm.", Insert: "norm()", Example: "norm(this.vector)"},
	{Domain: "numeric", Name: "normalize", Signature: "normalize(values)", Summary: "L2-normalize a numeric array.", Insert: "normalize()", Example: "normalize(this.vector)"},
	{Domain: "numeric", Name: "distance", Signature: "distance(x, y, method?)", Summary: "Numeric distance.", Insert: "distance()", Example: "distance(this.a, this.b, \"manhattan\")"},
	{Domain: "numeric", Name: "similarity", Signature: "similarity(x, y, method)", Summary: "Cosine, Jaccard, Hamming, or Levenshtein similarity.", Insert: "similarity()", Example: "similarity(this.a, this.b, \"cosine\")"},
	{Domain: "collection", Name: "chunk", Signature: "chunk(values, size)", Summary: "Split into fixed-size chunks.", Insert: "chunk()", Example: "chunk(this.items, 10)"},
	{Domain: "collection", Name: "zip", Signature: "zip(left, right)", Summary: "Zip equal-length arrays.", Insert: "zip()", Example: "zip(this.keys, this.values)"},
	{Domain: "collection", Name: "merge", Signature: "merge(left, right)", Summary: "Shallow map merge; right wins.", Insert: "merge()", Example: "merge(this.defaults, this.options)"},
	{Domain: "collection", Name: "union", Signature: "union(left, right)", Summary: "Stable unique union.", Insert: "union()", Example: "union(this.a, this.b)"},
	{Domain: "collection", Name: "intersection", Signature: "intersection(left, right)", Summary: "Stable unique intersection.", Insert: "intersection()", Example: "intersection(this.a, this.b)"},
	{Domain: "collection", Name: "difference", Signature: "difference(left, right)", Summary: "Stable unique difference.", Insert: "difference()", Example: "difference(this.a, this.b)"},
	{Domain: "collection", Name: "lag", Signature: "lag(values, periods)", Summary: "Lag values with leading nulls.", Insert: "lag()", Example: "lag(this.values, 1)"},
	{Domain: "collection", Name: "cumsum", Signature: "cumsum(values)", Summary: "Cumulative numeric sum.", Insert: "cumsum()", Example: "cumsum(this.values)"},
	{Domain: "collection", Name: "diff", Signature: "diff(values)", Summary: "Adjacent numeric differences.", Insert: "diff()", Example: "diff(this.values)"},
	{Domain: "text", Name: "regexFind", Signature: "regexFind(value, pattern)", Summary: "Find one RE2 match and groups.", Insert: "regexFind()", Example: "regexFind(this.text, \"(id-[0-9]+)\")"},
	{Domain: "text", Name: "regexFindAll", Signature: "regexFindAll(value, pattern)", Summary: "Find all RE2 matches and groups.", Insert: "regexFindAll()", Example: "regexFindAll(this.text, \"id-[0-9]+\")"},
	{Domain: "text", Name: "regexReplace", Signature: "regexReplace(value, pattern, replacement)", Summary: "Replace with RE2 capture expansion.", Insert: "regexReplace()", Example: "regexReplace(this.text, \"foo\", \"bar\")"},
	{Domain: "text", Name: "normalizeUnicode", Signature: "normalizeUnicode(value, form?)", Summary: "Normalize Unicode as NFC, NFD, NFKC, or NFKD.", Insert: "normalizeUnicode()", Example: "normalizeUnicode(this.text)"},
	{Domain: "text", Name: "removeDiacritics", Signature: "removeDiacritics(value)", Summary: "Remove combining marks and return NFC.", Insert: "removeDiacritics()", Example: "removeDiacritics(this.name)"},
	{Domain: "encoding", Name: "hash", Signature: "hash(value, algorithm)", Summary: "Hash exact strings or canonical JSON.", Insert: "hash()", Example: "hash(this.id, \"sha256\")"},
	{Domain: "encoding", Name: "checksum", Signature: "checksum(value, \"crc32\")", Summary: "CRC32 IEEE checksum.", Insert: "checksum()", Example: "checksum(this.payload, \"crc32\")"},
	{Domain: "encoding", Name: "hexEncode", Signature: "hexEncode(value)", Summary: "Encode UTF-8 text as lowercase hexadecimal.", Insert: "hexEncode()", Example: "hexEncode(this.id)"},
	{Domain: "encoding", Name: "hexDecode", Signature: "hexDecode(value)", Summary: "Decode lowercase or uppercase hexadecimal UTF-8 text.", Insert: "hexDecode()", Example: "hexDecode(\"6869\")"},
	{Domain: "encoding", Name: "urlEncode", Signature: "urlEncode(value, mode)", Summary: "URL-encode text in query or path_segment mode.", Insert: "urlEncode()", Example: "urlEncode(this.query, \"query\")"},
	{Domain: "encoding", Name: "urlDecode", Signature: "urlDecode(value, mode)", Summary: "URL-decode text in query or path_segment mode.", Insert: "urlDecode()", Example: "urlDecode(this.query, \"query\")"},
	{Domain: "network", Name: "normalizeIP", Signature: "normalizeIP(value)", Summary: "Normalize IPv4 or IPv6 text.", Insert: "normalizeIP()", Example: "normalizeIP(this.ip)"},
	{Domain: "network", Name: "inCIDR", Signature: "inCIDR(ip, cidr)", Summary: "Test IP membership without network access.", Insert: "inCIDR()", Example: "inCIDR(this.ip, \"10.0.0.0/8\")"},
	{Domain: "network", Name: "normalizeURL", Signature: "normalizeURL(value)", Summary: "Normalize an absolute URL without reordering its query.", Insert: "normalizeURL()", Example: "normalizeURL(this.url)"},
	{Domain: "network", Name: "normalizeHostname", Signature: "normalizeHostname(value)", Summary: "Normalize IDNA hostname text.", Insert: "normalizeHostname()", Example: "normalizeHostname(this.host)"},
	{Domain: "assignment", Name: "bucket", Signature: "bucket(value, count, namespace?)", Summary: "Deterministic XXH3 bucket.", Insert: "bucket()", Example: "bucket(this.user_id, 100)"},
	{Domain: "validate", Name: "is", Signature: "is(value, rule, args...)", Summary: "Apply a named pure validation rule.", Insert: "is()", Example: "is(this.email, \"email\")"},
}

// functionDocs contains the details shown in the workbench. Keep these
// descriptions concrete: an author should be able to use a callable without
// leaving the editor to look up its argument rules or return shape.
var functionDocs = map[string]functionDoc{
	"sqrt": {
		description: "Calculates the square root of one finite, non-negative number.",
		usage:       "Pass a number; negative, NaN, and infinite values return an evaluation error.",
		returns:     "A finite number.",
		notes:       "Use norm(values) for the length of a vector.",
	},
	"exp": {
		description: "Raises Euler's number (e) to the supplied exponent.",
		usage:       "Pass one finite number. Overflow is reported instead of returning infinity.",
		returns:     "A finite number.",
	},
	"clamp": {
		description: "Constrains a value to an inclusive lower and upper bound.",
		usage:       "clamp(value, min, max); all three arguments must be finite and min must not exceed max.",
		returns:     "The original value, min, or max, whichever is in range.",
	},
	"roundTo": {
		description: "Rounds a number to a chosen number of decimal places.",
		usage:       "roundTo(value, places), where places is an integer from -15 through 15 (negative values round to tens, hundreds, and so on).",
		returns:     "A finite rounded number.",
		notes:       "Rounding follows Go's math.Round behavior.",
	},
	"log": {
		description: "Calculates a logarithm. The natural logarithm is used when no base is supplied.",
		usage:       "log(value) or log(value, base); value and base must be finite, positive, and a base cannot be 1.",
		returns:     "A finite number.",
	},
	"variance": {
		description: "Measures how far numeric observations spread from their mean.",
		usage:       "variance(values) uses population variance; pass \"sample\" or \"population\" as the second argument to choose explicitly.",
		returns:     "A non-negative number. The input must be non-empty and finite.",
		notes:       "Sample variance requires at least two observations.",
	},
	"stddev": {
		description: "Returns the standard deviation of numeric observations.",
		usage:       "stddev(values) uses population behavior; use stddev(values, \"sample\") for sample data.",
		returns:     "The square root of variance.",
		notes:       "Sample standard deviation requires at least two observations.",
	},
	"quantile": {
		description: "Finds a percentile using sorted values and linear interpolation.",
		usage:       "quantile(values, p), with p between 0 and 1 inclusive (0.5 is the median).",
		returns:     "A number at the requested percentile; the input order is not changed.",
	},
	"covariance": {
		description: "Measures whether two numeric series move together.",
		usage:       "covariance(x, y) uses population behavior; add \"sample\" or \"population\" as the third argument.",
		returns:     "A number. Both finite arrays must be non-empty and the same length.",
		notes:       "Sample covariance requires at least two paired observations.",
	},
	"correlation": {
		description: "Measures the strength and direction of association between two series.",
		usage:       "correlation(x, y) uses Pearson correlation; pass \"spearman\" as the third argument for rank correlation.",
		returns:     "A number from -1 to 1.",
		notes:       "Both arrays must have equal length, finite values, and non-zero variance.",
	},
	"dot": {
		description: "Multiplies paired vector components and adds the products.",
		usage:       "dot(left, right) with two equal-length arrays of finite numbers.",
		returns:     "One number.",
	},
	"norm": {
		description: "Calculates the Euclidean (L2) length of a numeric vector.",
		usage:       "norm(values) with a non-empty array of finite numbers.",
		returns:     "A non-negative number.",
	},
	"normalize": {
		description: "Scales a numeric vector to unit Euclidean (L2) length.",
		usage:       "normalize(values) with a non-zero, non-empty array of finite numbers.",
		returns:     "A new array whose L2 norm is 1.",
		notes:       "A zero vector cannot be normalized.",
	},
	"distance": {
		description: "Computes a distance between numeric vectors or Unicode strings.",
		usage:       "distance(left, right) defaults to Euclidean; choose \"manhattan\" or \"chebyshev\" for vectors, \"hamming\" for equal-length arrays or strings, or \"levenshtein\" for strings.",
		returns:     "A non-negative number.",
		notes:       "Numeric vectors must be finite, non-empty, and equal in length. String comparisons use Unicode runes.",
	},
	"similarity": {
		description: "Compares two values with cosine, Jaccard, Hamming, or Levenshtein similarity.",
		usage:       "similarity(left, right, method); use cosine for numeric vectors, jaccard for arrays, or hamming/levenshtein for text.",
		returns:     "A number where 1 means identical for cosine/Jaccard/Levenshtein; Hamming returns the differing-position count.",
		notes:       "Cosine requires non-zero equal-length numeric vectors; Levenshtein compares Unicode strings.",
	},
	"chunk": {
		description: "Splits an array into consecutive chunks of a fixed size.",
		usage:       "chunk(values, size), where size is a positive integer. The final chunk may be shorter.",
		returns:     "An array of arrays in the original order.",
	},
	"zip": {
		description: "Pairs items at the same index from two arrays.",
		usage:       "zip(left, right) with equal-length arrays.",
		returns:     "An array of two-item arrays.",
	},
	"merge": {
		description: "Creates a shallow object containing keys from both inputs.",
		usage:       "merge(defaults, overrides); when a key appears twice, the right-hand value wins.",
		returns:     "A new object; nested objects are not recursively merged.",
	},
	"union": {
		description: "Combines two arrays into a stable set without duplicates.",
		usage:       "union(left, right); values keep their first-seen order and Expr-compatible numeric equality.",
		returns:     "A unique array.",
	},
	"intersection": {
		description: "Keeps values present in both arrays, once each.",
		usage:       "intersection(left, right); output follows the first array's order.",
		returns:     "A stable unique array.",
	},
	"difference": {
		description: "Keeps values from the first array that are absent from the second.",
		usage:       "difference(left, right); output follows the first array's order.",
		returns:     "A stable unique array.",
	},
	"lag": {
		description: "Moves a time series backward by a number of periods.",
		usage:       "lag(values, periods), where periods is non-negative. Leading positions are filled with null.",
		returns:     "An array with the same length as values.",
	},
	"cumsum": {
		description: "Builds a running sum from a numeric time series.",
		usage:       "cumsum(values) with finite numeric items; an empty array is allowed.",
		returns:     "An array of running totals with the same length as the input.",
	},
	"diff": {
		description: "Calculates each adjacent difference in a numeric time series.",
		usage:       "diff(values) subtracts the previous item from each following item; empty and one-item arrays are allowed.",
		returns:     "An array one item shorter than values.",
	},
	"regexFind": {
		description: "Finds the first RE2 regular-expression match and its capture groups.",
		usage:       "regexFind(text, pattern); the result starts with the whole match, followed by each capture group.",
		returns:     "An array of strings, or null when there is no match.",
		notes:       "Patterns are limited to 8 KiB and use Go RE2 syntax.",
	},
	"regexFindAll": {
		description: "Finds every non-overlapping RE2 match and its capture groups.",
		usage:       "regexFindAll(text, pattern) with a Go RE2 pattern.",
		returns:     "An array of match arrays; no matches returns an empty array.",
		notes:       "Patterns are limited to 8 KiB.",
	},
	"regexReplace": {
		description: "Replaces every RE2 match, including capture expansion in the replacement.",
		usage:       "regexReplace(text, pattern, replacement); use $1 or ${name} to insert captured groups.",
		returns:     "The resulting string.",
		notes:       "Patterns are limited to 8 KiB.",
	},
	"normalizeUnicode": {
		description: "Converts text to a Unicode normalization form.",
		usage:       "normalizeUnicode(text) defaults to NFC; choose \"nfc\", \"nfd\", \"nfkc\", or \"nfkd\" as the second argument.",
		returns:     "Normalized Unicode text.",
	},
	"removeDiacritics": {
		description: "Removes Unicode combining marks such as accents from text.",
		usage:       "removeDiacritics(text) for search keys or display-safe comparisons.",
		returns:     "NFC text without combining marks.",
		notes:       "This is a comparison aid, not a transliteration system.",
	},
	"hash": {
		description: "Hashes text or a JSON value with a named deterministic algorithm.",
		usage:       "hash(value, algorithm), using md5, sha1, sha256, sha384, sha512, or xxh3.",
		returns:     "Lowercase hexadecimal without a prefix; xxh3 is exactly 16 characters.",
		notes:       "Strings use their exact UTF-8 bytes. Other values use canonical JSON. MD5 and SHA-1 are compatibility hashes, not security algorithms.",
	},
	"checksum": {
		description: "Calculates a CRC32 IEEE checksum for text or a JSON value.",
		usage:       "checksum(value, \"crc32\"). Other algorithm names are rejected.",
		returns:     "Exactly eight lowercase hexadecimal characters.",
		notes:       "Strings use exact UTF-8 bytes; other values use canonical JSON.",
	},
	"hexEncode": {
		description: "Encodes UTF-8 text as hexadecimal bytes.",
		usage:       "hexEncode(text).",
		returns:     "Lowercase hexadecimal text.",
	},
	"hexDecode": {
		description: "Decodes hexadecimal bytes back to UTF-8 text.",
		usage:       "hexDecode(hex); input must contain an even number of valid hexadecimal digits and decode as UTF-8.",
		returns:     "Decoded text.",
	},
	"urlEncode": {
		description: "Escapes text for a URL component.",
		usage:       "urlEncode(value, mode), where mode is explicitly \"query\" or \"path_segment\".",
		returns:     "Percent-encoded text suitable for the selected component.",
	},
	"urlDecode": {
		description: "Decodes a URL component using the selected mode.",
		usage:       "urlDecode(value, mode), with mode \"query\" or \"path_segment\".",
		returns:     "Decoded UTF-8 text.",
	},
	"normalizeIP": {
		description: "Parses and prints an IPv4 or IPv6 address in its canonical text form.",
		usage:       "normalizeIP(value); no DNS lookup or network access is performed.",
		returns:     "Canonical IP text.",
	},
	"inCIDR": {
		description: "Checks whether an IP address belongs to a CIDR network.",
		usage:       "inCIDR(ip, cidr), for example inCIDR(this.ip, \"10.0.0.0/8\").",
		returns:     "true or false.",
		notes:       "Both inputs are parsed locally; no network access is performed.",
	},
	"normalizeURL": {
		description: "Normalizes an absolute URL without changing query ordering or the fragment.",
		usage:       "normalizeURL(value); the scheme and hostname become lowercase, IDNA is normalized, and default ports are removed.",
		returns:     "A normalized absolute URL.",
		notes:       "User information, path, query order, and fragment are preserved.",
	},
	"normalizeHostname": {
		description: "Normalizes a hostname to lowercase IDNA ASCII.",
		usage:       "normalizeHostname(value); one trailing root dot is removed.",
		returns:     "A canonical hostname.",
	},
	"bucket": {
		description: "Assigns a value deterministically to one of a fixed number of experiment buckets.",
		usage:       "bucket(value, count) or bucket(value, count, namespace); count is a positive integer.",
		returns:     "An integer from 0 through count-1.",
		notes:       "A namespace changes the hashed input to [namespace, value] and keeps assignments stable across evaluations.",
	},
	"is": {
		description: "Runs one named, pure validation predicate against a value.",
		usage:       "is(value, \"rule_name\"); browse Validation rules below for available names.",
		returns:     "true or false. Unknown rules and unsupported arguments are evaluation errors.",
		notes:       "Wrong value types return false; predicates never access storage or the network.",
	},
	"json": {
		description: "Reads a JSON path from the current input value.",
		usage:       "json(path), where path uses the supported JSONPath syntax and is evaluated against this.",
		returns:     "The selected JSON value, or nil when the path does not match.",
		notes:       "This is a read-only projection; it does not access files, storage, or the network.",
	},
	"raw": {
		description: "Serializes an expression value as JSON text.",
		usage:       "raw(value) when a downstream field needs the JSON representation rather than the value itself.",
		returns:     "A JSON string, subject to the expression output limit.",
	},
	"time": {
		description: "Converts a Unix timestamp into a time value.",
		usage:       "time(value), where value is Unix seconds or Unix nanoseconds.",
		returns:     "A UTC time value rendered as RFC 3339 in the evaluator response.",
	},
	"now": {
		description: "Returns the instant captured when this evaluation started.",
		usage:       "now() takes no arguments; repeated calls in one evaluation return the same instant.",
		returns:     "A UTC time value rendered as RFC 3339 in the evaluator response.",
		notes:       "A supplied request timestamp is normalized to UTC; otherwise the evaluator captures time once.",
	},
	"duration": {
		description: "Parses Go duration text into a duration value.",
		usage:       "duration(value), for example duration(\"5m\") or duration(\"250ms\").",
		returns:     "A duration value rendered with Go duration formatting in the evaluator response.",
	},
}

var coreFunctions = []Function{
	{Domain: "core", Name: "json", Signature: "json(path)", Summary: "Read a JSON path from this.", Insert: "json()", Example: "json(\"items.#(state==\\\"done\\\")\")"},
	{Domain: "core", Name: "raw", Signature: "raw(value)", Summary: "Return JSON text.", Insert: "raw()", Example: "raw(this.payload)"},
	{Domain: "core", Name: "time", Signature: "time(value)", Summary: "Convert seconds or nanoseconds to time.", Insert: "time()", Example: "time(this.timestamp)"},
	{Domain: "core", Name: "now", Signature: "now()", Summary: "Return the captured evaluation instant.", Insert: "now()", Example: "now()"},
	{Domain: "core", Name: "duration", Signature: "duration(value)", Summary: "Parse Go duration text.", Insert: "duration()", Example: "duration(\"5m\")"},
}

var builtinDocs = map[string]functionDoc{
	"all":           {signature: "all(values, predicate)", summary: "Checks that every item satisfies a predicate.", description: "Returns true when every item in a collection passes the predicate.", usage: "all(values, {.active}) or values.all({.active})", returns: "Boolean; an empty collection is true."},
	"none":          {signature: "none(values, predicate)", summary: "Checks that no item satisfies a predicate.", description: "Returns true when no item in a collection passes the predicate.", usage: "none(values, {.blocked}) or values.none({.blocked})", returns: "Boolean; an empty collection is true."},
	"any":           {signature: "any(values, predicate)", summary: "Checks that at least one item satisfies a predicate.", description: "Returns true when at least one item passes the predicate.", usage: "any(values, {.active}) or values.any({.active})", returns: "Boolean; an empty collection is false."},
	"one":           {signature: "one(values, predicate)", summary: "Checks that exactly one item satisfies a predicate.", description: "Returns true when exactly one item passes the predicate.", usage: "one(values, {.primary}) or values.one({.primary})", returns: "Boolean."},
	"filter":        {signature: "filter(values, predicate)", summary: "Keeps items that satisfy a predicate.", description: "Selects items for which the predicate is true.", usage: "filter(this.items, {.enabled})", returns: "A new array in input order."},
	"map":           {signature: "map(values, transform)", summary: "Transforms each item in a collection.", description: "Evaluates a transform for each item.", usage: "map(this.items, {.name})", returns: "A new array of transformed values."},
	"find":          {signature: "find(values, predicate)", summary: "Finds the first matching item.", description: "Returns the first item whose predicate is true.", usage: "find(this.items, {.id == \"target\"})", returns: "The item, or nil when no item matches."},
	"findIndex":     {signature: "findIndex(values, predicate)", summary: "Finds the first matching index.", description: "Returns the zero-based index of the first matching item.", usage: "findIndex(this.items, {.id == \"target\"})", returns: "An integer, or -1 when no item matches."},
	"findLast":      {signature: "findLast(values, predicate)", summary: "Finds the last matching item.", description: "Returns the last item whose predicate is true.", usage: "findLast(this.items, {.active})", returns: "The item, or nil when no item matches."},
	"findLastIndex": {signature: "findLastIndex(values, predicate)", summary: "Finds the last matching index.", description: "Returns the zero-based index of the last matching item.", usage: "findLastIndex(this.items, {.active})", returns: "An integer, or -1 when no item matches."},
	"count":         {signature: "count(values, predicate)", summary: "Counts items that satisfy a predicate.", description: "Counts how many items pass the predicate.", usage: "count(this.items, {.state == \"ready\"})", returns: "A non-negative integer."},
	"sum":           {signature: "sum(values, predicate)", summary: "Sums values selected by a predicate.", description: "Adds numeric items, optionally selecting them with a predicate.", usage: "sum(this.items, {.amount})", returns: "A number."},
	"groupBy":       {signature: "groupBy(values, key)", summary: "Groups items by a computed key.", description: "Builds a map from each key to the items that produced it.", usage: "groupBy(this.orders, {.status})", returns: "An object whose values are arrays."},
	"sortBy":        {signature: "sortBy(values, key, order?)", summary: "Sorts items by a computed key.", description: "Returns items ordered by a key expression.", usage: "sortBy(this.orders, {.created_at}, \"asc\")", returns: "A new sorted array."},
	"reduce":        {signature: "reduce(values, reducer, initial)", summary: "Folds a collection into one value.", description: "Carries an accumulator through the collection.", usage: "reduce(this.values, {# + #}, 0)", returns: "The final accumulator value."},
	"len":           {signature: "len(value)", summary: "Counts items, keys, or Unicode characters.", description: "Returns the length of an array, map, or string (strings count Unicode runes).", usage: "len(this.items)", returns: "An integer."},
	"type":          {signature: "type(value)", summary: "Reports the runtime value type.", description: "Returns a stable type label such as string, int, float, array, map, or nil.", usage: "type(this.value)", returns: "A string."},
	"abs":           {signature: "abs(value)", summary: "Returns an absolute numeric value.", description: "Removes the sign from an integer or floating-point value.", usage: "abs(this.delta)", returns: "A number with no negative sign."},
	"ceil":          {signature: "ceil(value)", summary: "Rounds a number upward.", description: "Rounds toward positive infinity.", usage: "ceil(this.score)", returns: "A number."},
	"floor":         {signature: "floor(value)", summary: "Rounds a number downward.", description: "Rounds toward negative infinity.", usage: "floor(this.score)", returns: "A number."},
	"round":         {signature: "round(value)", summary: "Rounds a number to the nearest integer.", description: "Rounds a numeric value using Go's math.Round behavior.", usage: "round(this.score)", returns: "A number."},
	"int":           {signature: "int(value)", summary: "Converts a value to an integer.", description: "Converts numeric values or numeric strings to an integer.", usage: "int(this.count)", returns: "An integer; invalid conversions fail."},
	"float":         {signature: "float(value)", summary: "Converts a value to a floating-point number.", description: "Converts numeric values or numeric strings to a float.", usage: "float(this.ratio)", returns: "A number; invalid conversions fail."},
	"string":        {signature: "string(value)", summary: "Formats a value as text.", description: "Formats a value using its standard textual representation.", usage: "string(this.id)", returns: "A string."},
	"trim":          {signature: "trim(text, cutset?)", summary: "Removes whitespace or a cutset from both ends.", description: "Trims spaces by default, or characters in the optional cutset.", usage: "trim(this.name)", returns: "A string."},
	"trimPrefix":    {signature: "trimPrefix(text, prefix?)", summary: "Removes a prefix when present.", description: "Removes the supplied prefix, defaulting to one space.", usage: "trimPrefix(this.path, \"/api\")", returns: "A string."},
	"trimSuffix":    {signature: "trimSuffix(text, suffix?)", summary: "Removes a suffix when present.", description: "Removes the supplied suffix, defaulting to one space.", usage: "trimSuffix(this.name, \".tmp\")", returns: "A string."},
	"upper":         {signature: "upper(text)", summary: "Converts text to uppercase.", description: "Uppercases Unicode text.", usage: "upper(this.code)", returns: "A string."},
	"lower":         {signature: "lower(text)", summary: "Converts text to lowercase.", description: "Lowercases Unicode text.", usage: "lower(this.email)", returns: "A string."},
	"split":         {signature: "split(text, separator, limit?)", summary: "Splits text into pieces.", description: "Splits on a separator, optionally limiting the number of pieces.", usage: "split(this.tags, \",\")", returns: "An array of strings."},
	"splitAfter":    {signature: "splitAfter(text, separator, limit?)", summary: "Splits text while retaining separators.", description: "Splits after each separator, optionally limiting pieces.", usage: "splitAfter(this.path, \"/\")", returns: "An array of strings."},
	"replace":       {signature: "replace(text, old, new, count?)", summary: "Replaces text occurrences.", description: "Replaces all occurrences or a chosen count when supplied.", usage: "replace(this.name, \"old\", \"new\")", returns: "A string."},
	"repeat":        {signature: "repeat(text, count)", summary: "Repeats text a bounded number of times.", description: "Repeats a string count times.", usage: "repeat(\"-\", 10)", returns: "A string.", notes: "The VM memory limit bounds the resulting allocation."},
	"join":          {signature: "join(values, separator?)", summary: "Joins strings with a separator.", description: "Concatenates an array of strings.", usage: "join(this.tags, \", \")", returns: "A string."},
	"indexOf":       {signature: "indexOf(text, search)", summary: "Finds the first substring index.", description: "Returns the zero-based index of the first occurrence.", usage: "indexOf(this.text, \"error\")", returns: "An integer, or -1."},
	"lastIndexOf":   {signature: "lastIndexOf(text, search)", summary: "Finds the last substring index.", description: "Returns the zero-based index of the last occurrence.", usage: "lastIndexOf(this.path, \"/\")", returns: "An integer, or -1."},
	"hasPrefix":     {signature: "hasPrefix(text, prefix)", summary: "Checks a text prefix.", description: "Tests whether text starts with prefix.", usage: "hasPrefix(this.path, \"/api\")", returns: "Boolean."},
	"hasSuffix":     {signature: "hasSuffix(text, suffix)", summary: "Checks a text suffix.", description: "Tests whether text ends with suffix.", usage: "hasSuffix(this.file, \".json\")", returns: "Boolean."},
	"max":           {signature: "max(values...)", summary: "Returns the largest numeric value.", description: "Finds the maximum across values or nested collections.", usage: "max(this.scores)", returns: "A number; empty input returns the zero value."},
	"min":           {signature: "min(values...)", summary: "Returns the smallest numeric value.", description: "Finds the minimum across values or nested collections.", usage: "min(this.scores)", returns: "A number; empty input returns the zero value."},
	"mean":          {signature: "mean(values...)", summary: "Returns the arithmetic mean.", description: "Calculates the average of numeric values or nested collections.", usage: "mean(this.scores)", returns: "A number; empty input returns 0."},
	"median":        {signature: "median(values...)", summary: "Returns the median value.", description: "Sorts numeric values and returns the middle value (or middle pair average).", usage: "median(this.scores)", returns: "A number; empty input returns 0."},
	"toJSON":        {signature: "toJSON(value)", summary: "Formats a value as indented JSON.", description: "Serializes a value for display or transport.", usage: "toJSON(this.payload)", returns: "Indented JSON text."},
	"fromJSON":      {signature: "fromJSON(text)", summary: "Parses JSON text.", description: "Decodes a JSON string into an expression value.", usage: "fromJSON(this.payload)", returns: "The parsed JSON value."},
	"toBase64":      {signature: "toBase64(text)", summary: "Encodes text as Base64.", description: "Encodes UTF-8 text using standard Base64.", usage: "toBase64(this.secret_id)", returns: "Base64 text."},
	"fromBase64":    {signature: "fromBase64(text)", summary: "Decodes Base64 text.", description: "Decodes standard Base64 into UTF-8 text.", usage: "fromBase64(this.payload)", returns: "Decoded text."},
	"date":          {signature: "date(value, layout?, timezone?)", summary: "Parses a date or time string.", description: "Parses common date formats or a Go layout, optionally in a named timezone.", usage: "date(this.created_at, time.RFC3339)", returns: "A time value."},
	"timezone":      {signature: "timezone(name)", summary: "Loads a named timezone.", description: "Resolves an IANA timezone name for date and time operations.", usage: "timezone(\"Asia/Dubai\")", returns: "A timezone value; invalid names fail."},
	"first":         {signature: "first(values)", summary: "Returns the first item.", description: "Gets the first element of an array.", usage: "first(this.items)", returns: "The item, or nil for an empty array."},
	"last":          {signature: "last(values)", summary: "Returns the last item.", description: "Gets the last element of an array.", usage: "last(this.items)", returns: "The item, or nil for an empty array."},
	"get":           {signature: "get(value, path...)", summary: "Reads a nested value by path.", description: "Accesses a nested map or array value with a dynamic path.", usage: "get(this, \"user\", \"email\")", returns: "The selected value, or nil when absent."},
	"take":          {signature: "take(values, count)", summary: "Takes the first N items.", description: "Returns up to count items from the beginning of an array.", usage: "take(this.items, 10)", returns: "A new array."},
	"keys":          {signature: "keys(object)", summary: "Lists object keys.", description: "Returns the keys of a map.", usage: "keys(this.attributes)", returns: "An array of keys."},
	"values":        {signature: "values(object)", summary: "Lists object values.", description: "Returns the values of a map.", usage: "values(this.attributes)", returns: "An array of values."},
	"toPairs":       {signature: "toPairs(object)", summary: "Converts an object to key-value pairs.", description: "Represents each map entry as a two-item pair.", usage: "toPairs(this.attributes)", returns: "An array of pairs."},
	"fromPairs":     {signature: "fromPairs(pairs)", summary: "Builds an object from key-value pairs.", description: "Converts two-item pairs into a map.", usage: "fromPairs([[\"state\", \"ready\"]])", returns: "An object."},
	"reverse":       {signature: "reverse(values)", summary: "Reverses an array.", description: "Returns items in reverse order without changing the input.", usage: "reverse(this.items)", returns: "A new array."},
	"uniq":          {signature: "uniq(values)", summary: "Removes duplicate array items.", description: "Keeps the first occurrence of each value.", usage: "uniq(this.tags)", returns: "A stable unique array."},
	"concat":        {signature: "concat(values...)", summary: "Concatenates arrays.", description: "Appends any number of arrays into one array.", usage: "concat(this.pending, this.complete)", returns: "A new array."},
	"flatten":       {signature: "flatten(values)", summary: "Flattens nested arrays.", description: "Recursively removes array nesting.", usage: "flatten(this.batches)", returns: "A flat array."},
	"sort":          {signature: "sort(values, order?)", summary: "Sorts an array.", description: "Sorts arrays of comparable values; order defaults to ascending.", usage: "sort(this.scores, \"desc\")", returns: "A new sorted array."},
	"bitnot":        {signature: "bitnot(value)", summary: "Inverts integer bits.", description: "Applies bitwise NOT to an integer.", usage: "bitnot(this.flags)", returns: "An integer."},
	"bitand":        {signature: "bitand(left, right)", summary: "Applies bitwise AND.", description: "Keeps bits set in both integers.", usage: "bitand(this.flags, 4)", returns: "An integer."},
	"bitor":         {signature: "bitor(left, right)", summary: "Applies bitwise OR.", description: "Sets bits present in either integer.", usage: "bitor(this.flags, 4)", returns: "An integer."},
	"bitxor":        {signature: "bitxor(left, right)", summary: "Applies bitwise XOR.", description: "Sets bits present in exactly one integer.", usage: "bitxor(this.flags, 4)", returns: "An integer."},
	"bitnand":       {signature: "bitnand(left, right)", summary: "Applies bit clear (AND NOT).", description: "Clears right-hand bits from the left integer.", usage: "bitnand(this.flags, 4)", returns: "An integer."},
	"bitshl":        {signature: "bitshl(value, shift)", summary: "Shifts integer bits left.", description: "Moves bits left by a non-negative count.", usage: "bitshl(this.flags, 2)", returns: "An integer."},
	"bitshr":        {signature: "bitshr(value, shift)", summary: "Shifts signed integer bits right.", description: "Moves bits right by a non-negative count while preserving the sign.", usage: "bitshr(this.flags, 2)", returns: "An integer."},
	"bitushr":       {signature: "bitushr(value, shift)", summary: "Shifts integer bits right without sign extension.", description: "Moves bits right by a non-negative count and fills with zeroes.", usage: "bitushr(this.flags, 2)", returns: "An integer."},
}

func applyFunctionDoc(item Function) Function {
	doc, ok := functionDocs[item.Name]
	if !ok {
		doc, ok = builtinDocs[item.Name]
	}
	if !ok {
		return item
	}
	if doc.signature != "" {
		item.Signature = doc.signature
	}
	if doc.summary != "" {
		item.Summary = doc.summary
	}
	item.Description = doc.description
	item.Usage = doc.usage
	item.Returns = doc.returns
	item.Notes = doc.notes
	return item
}

// Functions returns the stable language catalog.
func Functions() []Function {
	out := append([]Function(nil), customFunctions...)
	out = append(out, coreFunctions...)
	for i := range out {
		out[i] = applyFunctionDoc(out[i])
	}
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[item.Name] = struct{}{}
	}
	for _, item := range builtin.Builtins {
		if _, ok := seen[item.Name]; ok {
			continue
		}
		out = append(out, applyFunctionDoc(Function{
			Domain:    "expr",
			Name:      item.Name,
			Signature: item.Name + "(...)",
			Summary:   "Built-in expression callable.",
			Insert:    item.Name + "()",
			Example:   item.Name + "(...)",
		}))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Domain == out[j].Domain {
			return out[i].Name < out[j].Name
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

// Guide returns the embedded language guide.
func Guide() string {
	body, _ := referenceFS.ReadFile("reference.md")
	return string(body)
}

// GetReference returns the complete reference payload.
func GetReference() Reference {
	return Reference{Version: "1.17.8", Upstream: "https://expr-lang.org/docs/language-definition", Guide: Guide(), Functions: Functions(), Rules: validate.Rules()}
}
