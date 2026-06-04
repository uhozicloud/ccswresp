package main

import (
	"fmt"
	"os"
)

// Color codes for terminal output
const (
	cReset   = "\033[0m"
	cCyan    = "\033[36m"
	cGreen   = "\033[32m"
	cYellow  = "\033[33m"
	cRed     = "\033[31m"
	cMagenta = "\033[35m"
	cGray    = "\033[90m"
	cBold    = "\033[1m"
)

var quietMode bool

func logInfo(msg string, args ...interface{}) {
	fmt.Printf(cCyan+"[INFO]"+cReset+" "+msg+"\n", args...)
}

func logOk(msg string, args ...interface{}) {
	fmt.Printf(cGreen+"[ OK ]"+cReset+" "+msg+"\n", args...)
}

func logWarn(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, cYellow+"[WARN]"+cReset+" "+msg+"\n", args...)
}

func logErr(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, cRed+"[ERR ]"+cReset+" "+msg+"\n", args...)
}

func logReq(msg string, args ...interface{}) {
	if quietMode {
		return
	}
	fmt.Printf(cMagenta+"[REQ ]"+cReset+" "+msg+"\n", args...)
}

func logResp(msg string, args ...interface{}) {
	if quietMode {
		return
	}
	fmt.Printf(cGreen+"[RESP]"+cReset+" "+msg+"\n", args...)
}

func logSkip(msg string, args ...interface{}) {
	fmt.Printf(cGray+"[SKIP]"+cReset+" "+msg+"\n", args...)
}

func logToks(prompt, completion, total int) {
	parts := ""
	if prompt > 0 {
		parts += fmt.Sprintf("in:%d ", prompt)
	}
	if completion > 0 {
		parts += fmt.Sprintf("out:%d ", completion)
	}
	if total > 0 {
		parts += fmt.Sprintf("total:%d", total)
	}
	fmt.Printf(cGray+"[TOKS]"+cReset+" %s\n", parts)
}

func bold(s string) string {
	return cBold + s + cReset
}

func cyan(s string) string {
	return cCyan + s + cReset
}
