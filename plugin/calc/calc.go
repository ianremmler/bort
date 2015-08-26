// Package calc is a bort IRC bot plugin that provides an interface to the clac
// RPN calculator.
package calc

import (
	"errors"
	"fmt"
	"io"
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
		"div":    cl.IntDiv,
		"mod":    cl.Mod,
		"exp":    cl.Exp,
		"^":      cl.Pow,
		"2^":     cl.Pow2,
		"10^":    cl.Pow10,
		"logn":   cl.LogN,
		"ln":     cl.Ln,
		"log":    cl.Log,
		"lg":     cl.Lg,
		"sqrt":   cl.Sqrt,
		"!":      cl.Factorial,
		"comb":   cl.Comb,
		"perm":   cl.Perm,
		"sin":    cl.Sin,
		"cos":    cl.Cos,
		"tan":    cl.Tan,
		"asin":   cl.Asin,
		"acos":   cl.Acos,
		"atan":   cl.Atan,
		"atan2":  cl.Atan2,
		"dtor":   cl.DegToRad,
		"rtod":   cl.RadToDeg,
		"rtop":   cl.RectToPolar,
		"ptor":   cl.PolarToRect,
		"floor":  cl.Floor,
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
		"drop":   cl.Drop,
		"dropn":  cl.DropN,
		"dropr":  cl.DropR,
		"dup":    cl.Dup,
		"dupn":   cl.DupN,
		"dupr":   cl.DupR,
		"pick":   cl.Pick,
		"swap":   cl.Swap,
		"depth":  cl.Depth,
		"min":    cl.Min,
		"max":    cl.Max,
		"minn":   cl.MinN,
		"maxn":   cl.MaxN,
		"rot":    cl.Rot,
		"rotr":   cl.RotR,
		"unrot":  cl.Unrot,
		"unrotr": cl.UnrotR,
		"mag":    cl.Mag,
		"hyp":    cl.Hypot,
		"dot":    cl.Dot,
		"dot3":   cl.Dot3,
		"cross":  cl.Cross,
		"pi":     func() error { return cl.Push(clac.Pi) },
		"e":      func() error { return cl.Push(clac.E) },
		"phi":    func() error { return cl.Push(clac.Phi) },
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
	cmdReader := strings.NewReader(in.Args)
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

	if isHex {
		clac.SetFormat("%#x")
	} else {
		clac.SetFormat("%12g")
	}
	text := ""
	for i := range stack {
		val := stack[len(stack)-i-1]
		var err error
		if isHex {
			val, err = clac.Trunc(val)
		}
		if err != nil {
			text += "error"
		} else {
			text += fmt.Sprint(val)
		}
		if i < len(stack)-1 {
			text += " "
		}
	}
	out.Text = text
	return nil
}
