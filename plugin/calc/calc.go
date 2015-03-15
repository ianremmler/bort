// Package calc is a bort IRC bot plugin that provides an interface to the clac
// RPN calculator.
package calc

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/ianremmler/bort"
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
		"2^":     cl.Pow2,
		"10^":    cl.Pow10,
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
		"dtor":   cl.DegToRad,
		"rtod":   cl.RadToDeg,
		"floor":  cl.Floor,
		"rtop":   cl.RectToPolar,
		"ptor":   cl.PolarToRect,
		"ceil":   cl.Ceil,
		"trunc":  cl.Trunc,
		"and":    cl.And,
		"or":     cl.Or,
		"xor":    cl.Xor,
		"not":    cl.Not,
		"andn":   cl.AndN,
		"orn":    cl.OrN,
		"xorn":   cl.XorN,
		"sum":    cl.Sum,
		"avg":    cl.Avg,
		"dot":    cl.Dot,
		"dot3":   cl.Dot3,
		"cross":  cl.Cross,
		"mag":    cl.Mag,
		"clear":  cl.Clear,
		"drop":   cl.Drop,
		"dropn":  cl.DropN,
		"dropr":  cl.DropR,
		"dup":    cl.Dup,
		"dupn":   cl.DupN,
		"dupr":   cl.DupR,
		"pick":   cl.Pick,
		"swap":   cl.Swap,
		"depth":  cl.Depth,
		"undo":   cl.Undo,
		"redo":   cl.Redo,
		"min":    cl.Min,
		"max":    cl.Max,
		"minn":   cl.MinN,
		"maxn":   cl.MaxN,
		"rot":    cl.Rot,
		"rotr":   cl.RotR,
		"unrot":  cl.Unrot,
		"unrotr": cl.UnrotR,
		"pi":     func() error { return cl.Push(math.Pi) },
		"e":      func() error { return cl.Push(math.E) },
		"phi":    func() error { return cl.Push(math.Phi) },
	}
	helpStr string
)

func init() {
	cmdList := []string{"help"}
	for cmd := range cmdMap {
		cmdList = append(cmdList, cmd)
	}
	sort.Strings(cmdList)
	helpStr = strings.Join(cmdList, " ")
	bort.RegisterCommand("calc", "RPN calculator", Calc)
}

// Calc passes input to clac and returns the calculated result.
func Calc(in, out *bort.Message) error {
	out.Type = bort.PrivMsg
	cl.Reset()
	cmdReader := strings.NewReader(in.Text)
	isHex := false
	for {
		tok := ""
		if _, err := fmt.Fscan(cmdReader, &tok); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		switch tok { // special cases
		case "help":
			out.Target = in.Nick
			out.Text = helpStr
			return nil
		case "hex":
			isHex = true
			continue
		}
		num, err := clac.ParseNum(tok)
		if err == nil {
			if err = cl.Exec(func() error { return cl.Push(num) }); err != nil {
				return fmt.Errorf("push: %s", err)
			}
			continue
		}
		if cmd, ok := cmdMap[tok]; ok {
			if err = cl.Exec(cmd); err != nil {
				return fmt.Errorf("calc: %s: invalid input", tok)
			}
			continue
		}
		return fmt.Errorf("calc: %s: invalid input", tok)
	}
	stack := cl.Stack()
	if len(stack) == 0 {
		return errors.New("empty stack")
	}
	ans := ""
	for i := range stack {
		val := stack[len(stack)-i-1]
		if isHex {
			if math.Abs(val) >= 1<<53 {
				ans += "overflow"
			} else {
				ans += fmt.Sprintf("%#x", int64(val))
			}
		} else {
			ans += fmt.Sprintf("%g", val)
		}
		if i < len(stack)-1 {
			ans += " "
		}
	}
	out.Text = ans
	return nil
}
