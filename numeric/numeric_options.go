package numeric

import (
	"fmt"

	exprlib "github.com/expr-lang/expr"
)

// Options returns the Expr registrations for numeric and vector helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		fn("sqrt", sqrtFunc),
		fn("exp", expFunc),
		fn("clamp", clampFunc),
		fn("roundTo", roundToFunc),
		fn("log", logFunc),
		fn("variance", varianceFunc),
		fn("stddev", stddevFunc),
		fn("quantile", quantileFunc),
		fn("covariance", covarianceFunc),
		fn("correlation", correlationFunc),
		fn("dot", dotFunc),
		fn("norm", normFunc),
		fn("normalize", normalizeFunc),
		fn("distance", distanceFunc),
		fn("similarity", similarityFunc),
	}
}

func sqrtFunc(args ...any) (any, error) {
	v, err := oneNumber(args, "sqrt")
	if err != nil {
		return nil, err
	}
	return Sqrt(v)
}

func expFunc(args ...any) (any, error) {
	v, err := oneNumber(args, "exp")
	if err != nil {
		return nil, err
	}
	return Exp(v)
}

func clampFunc(args ...any) (any, error) {
	if err := arity(args, 3, "clamp"); err != nil {
		return nil, err
	}
	values, err := numberArgs(args[:3])
	if err != nil {
		return nil, err
	}
	return Clamp(values[0], values[1], values[2])
}

func roundToFunc(args ...any) (any, error) {
	if err := arity(args, 2, "roundTo"); err != nil {
		return nil, err
	}
	v, err := number(args[0])
	if err != nil {
		return nil, err
	}
	places, err := integer(args[1])
	if err != nil {
		return nil, err
	}
	return RoundTo(v, places)
}

func logFunc(args ...any) (any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("log: expected one or two arguments")
	}
	v, err := number(args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 1 {
		return Log(v)
	}
	base, err := number(args[1])
	if err != nil {
		return nil, err
	}
	return Log(v, base)
}

func varianceFunc(args ...any) (any, error) {
	values, method, err := statisticArgs(args, "variance")
	if err != nil {
		return nil, err
	}
	return Variance(values, method)
}

func stddevFunc(args ...any) (any, error) {
	values, method, err := statisticArgs(args, "stddev")
	if err != nil {
		return nil, err
	}
	return StdDev(values, method)
}

func quantileFunc(args ...any) (any, error) {
	if err := arity(args, 2, "quantile"); err != nil {
		return nil, err
	}
	values, err := numbers(args[0])
	if err != nil {
		return nil, err
	}
	p, err := number(args[1])
	if err != nil {
		return nil, err
	}
	return Quantile(values, p)
}

func covarianceFunc(args ...any) (any, error) {
	x, y, method, err := pairArgs(args, "covariance")
	if err != nil {
		return nil, err
	}
	return Covariance(x, y, method)
}

func correlationFunc(args ...any) (any, error) {
	x, y, method, err := pairArgs(args, "correlation")
	if err != nil {
		return nil, err
	}
	return Correlation(x, y, method)
}

func dotFunc(args ...any) (any, error) {
	if err := arity(args, 2, "dot"); err != nil {
		return nil, err
	}
	x, y, _, err := pairArgs(args, "dot")
	if err != nil {
		return nil, err
	}
	return Dot(x, y)
}

func normFunc(args ...any) (any, error) {
	if err := arity(args, 1, "norm"); err != nil {
		return nil, err
	}
	values, err := numbers(args[0])
	if err != nil {
		return nil, err
	}
	return Norm(values)
}

func normalizeFunc(args ...any) (any, error) {
	if err := arity(args, 1, "normalize"); err != nil {
		return nil, err
	}
	values, err := numbers(args[0])
	if err != nil {
		return nil, err
	}
	return Normalize(values)
}

func distanceFunc(args ...any) (any, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("distance: expected two or three arguments")
	}
	method, err := optionalMethod(args[2:])
	if err != nil {
		return nil, err
	}
	return Distance(args[0], args[1], method)
}

func similarityFunc(args ...any) (any, error) {
	if err := arity(args, 3, "similarity"); err != nil {
		return nil, err
	}
	method, err := text(args[2])
	if err != nil {
		return nil, err
	}
	return Similarity(args[0], args[1], method)
}

func arity(args []any, want int, name string) error {
	if len(args) != want {
		return fmt.Errorf("%s: expected %d arguments", name, want)
	}
	return nil
}

func numberArgs(args []any) ([]float64, error) {
	values := make([]float64, len(args))
	for i, arg := range args {
		value, err := number(arg)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func pairArgs(args []any, name string) ([]float64, []float64, string, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, nil, "", fmt.Errorf("%s: expected two or three arguments", name)
	}
	x, err := numbers(args[0])
	if err != nil {
		return nil, nil, "", err
	}
	y, err := numbers(args[1])
	if err != nil {
		return nil, nil, "", err
	}
	method, err := optionalMethod(args[2:])
	return x, y, method, err
}

func optionalMethod(args []any) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	return text(args[0])
}

func fn(name string, call func(...any) (any, error)) exprlib.Option {
	return exprlib.Function(name, call, new(func(...any) any))
}

func statisticArgs(args []any, name string) ([]float64, string, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, "", fmt.Errorf("%s: expected one or two arguments", name)
	}
	values, err := numbers(args[0])
	if err != nil {
		return nil, "", err
	}
	method := ""
	if len(args) == 2 {
		method, err = text(args[1])
	}
	return values, method, err
}
