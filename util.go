package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Go is a basic promise implementation: it wraps calls a function in a goroutine
// and returns a channel which will later return the function's return value.
func Go(f func() error) chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

// if os.env DEBUG set, debug is on
func Debugf(format string, a ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "<unknown>"
			line = -1
		} else {
			file = filepath.Base(file)
		}
		fmt.Fprintf(os.Stderr, fmt.Sprintf("[debug] %s:%d %s\n", file, line, format), a...)
	}
}

const (
	Gray = uint8(iota + 90)
	Red
	Green
	Yellow
	Blue
	Magenta
	//NRed      = uint8(31) // Normal
	EndColor = "\033[0m"

	TRAC = "TRAC"
	ERRO = "ERRO"
	WARN = "WARN"
	SUCC = "SUCC"
)

// colorLog colors log and print to stdout.
// Log format: [<level>] <content [path]> [ error ].
// Level: ERRO -> red; WARN -> Magenta; SUCC -> green; others -> default.
// Content: default; path: yellow; error -> red.
// Errors have to surrounded by "[ " and " ]"(space).
func colorLog(format string, a ...interface{}) {
	log := fmt.Sprintf(format, a...)
	if len(log) == 0 {
		return
	}

	if runtime.GOOS != "windows" {
		var clog string

		// Level.
		i := strings.Index(log, "]")
		if log[0] == '[' && i > -1 {
			clog += "[" + getColorLevel(log[1:i]) + "]"
		}

		log = log[i+1:]

		// Error.
		log = strings.Replace(log, "[ ", fmt.Sprintf("[\033[%dm", Red), -1)
		log = strings.Replace(log, " ]", EndColor+"]", -1)

		// Path.
		log = strings.Replace(log, "( ", fmt.Sprintf("(\033[%dm", Yellow), -1)
		log = strings.Replace(log, " )", EndColor+")", -1)

		// Highlights.
		log = strings.Replace(log, "# ", fmt.Sprintf("\033[%dm", Gray), -1)
		log = strings.Replace(log, " #", EndColor, -1)

		log = clog + log
	}

	var currentTime = time.Now()

	fmt.Print(currentTime.Format("2006-01-02 03:04:05 "+log))
}

// getColorLevel returns colored level string by given level.
func getColorLevel(level string) string {
	level = strings.ToUpper(level)
	switch level {
	case TRAC:
		return fmt.Sprintf("\033[%dm%s\033[0m", Blue, level)
	case ERRO:
		return fmt.Sprintf("\033[%dm%s\033[0m", Red, level)
	case WARN:
		return fmt.Sprintf("\033[%dm%s\033[0m", Magenta, level)
	case SUCC:
		return fmt.Sprintf("\033[%dm%s\033[0m", Green, level)
	default:
		return level
	}
}
