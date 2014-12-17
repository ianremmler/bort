package calc

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/ianremmler/clac"
)

var (
	cl     = clac.New()
	cmdMap = map[string]func() error{
		"neg":    cl.Neg,
		"abs":    cl.Abs,
		"inv":    cl.Inv,
		"+":      cl.Add,
		"-":      cl.Sub,
		"*":      cl.Mul,
		"/":      cl.Div,
		"mod":    cl.Mod,
		"exp":    cl.Exp,
		"^":      cl.Pow,
		"^2":     cl.Pow2,
		"^10":    cl.Pow10,
		"ln":     cl.Ln,
		"log":    cl.Log,
		"lg":     cl.Lg,
		"sqrt":   cl.Sqrt,
		"hypot":  cl.Hypot,
		"gamma":  cl.Gamma,
		"!":      cl.Factorial,
		"comb":   cl.Comb,
		"perm":   cl.Perm,
		"sin":    cl.Sin,
		"cos":    cl.Cos,
		"tan":    cl.Tan,
		"asin":   cl.Asin,
		"acos":   cl.Acos,
		"atan":   cl.Atan,
		"sinh":   cl.Sin,
		"cosh":   cl.Cos,
		"tanh":   cl.Tan,
		"asinh":  cl.Asin,
		"acosh":  cl.Acos,
		"atanh":  cl.Atan,
		"atan2":  cl.Atan2,
		"d->r":   cl.DegToRad,
		"r->d":   cl.RadToDeg,
		"floor":  cl.Floor,
		"r->p":   cl.RectToPolar,
		"p->r":   cl.PolarToRect,
		"ceil":   cl.Ceil,
		"trunc":  cl.Trunc,
		"and":    cl.And,
		"or":     cl.Or,
		"xor":    cl.Xor,
		"not":    cl.Not,
		"andn":   cl.Andn,
		"orn":    cl.Orn,
		"xorn":   cl.Xorn,
		"sum":    cl.Sum,
		"avg":    cl.Avg,
		"clear":  cl.Clear,
		"drop":   cl.Drop,
		"dropn":  cl.Dropn,
		"dropr":  cl.Dropr,
		"dup":    cl.Dup,
		"dupn":   cl.Dupn,
		"dupr":   cl.Dupr,
		"pick":   cl.Pick,
		"swap":   cl.Swap,
		"depth":  cl.Depth,
		"undo":   cl.Undo,
		"redo":   cl.Redo,
		"min":    cl.Min,
		"max":    cl.Max,
		"minn":   cl.Minn,
		"maxn":   cl.Maxn,
		"rot":    cl.Rot,
		"rotr":   cl.Rotr,
		"unrot":  cl.Unrot,
		"unrotr": cl.Unrotr,
		"pi":     func() error { return cl.Push(math.Pi) },
		"e":      func() error { return cl.Push(math.E) },
		"phi":    func() error { return cl.Push(math.Phi) },
	}
)

func Calc(input string) (string, error) {
	cl.Reset()
	cmdReader := strings.NewReader(input)
	for {
		tok := ""
		if _, err := fmt.Fscan(cmdReader, &tok); err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}
		num, err := clac.ParseNum(tok)
		if err == nil {
			if err = cl.Exec(func() error { return cl.Push(num) }); err != nil {
				return "", errors.New("push: " + err.Error())
			}
			continue
		}
		if cmd, ok := cmdMap[tok]; ok {
			if err = cl.Exec(cmd); err != nil {
				return "", errors.New(tok + ": " + err.Error())
			}
			continue
		}
		return "", errors.New(tok + ": invalid input")
	}
	stack := cl.Stack()
	if len(stack) == 0 {
		return "", errors.New("empty stack")
	}
	ans := ""
	for i := range stack {
		val := stack[len(stack)-i-1]
		ans += fmt.Sprintf("%g", val)
		if math.Abs(val) < math.MaxInt64 {
			ans += fmt.Sprintf("(%#x)", int64(val))
		}
		if i < len(stack)-1 {
			ans += " "
		}
	}
	return ans, nil
}
