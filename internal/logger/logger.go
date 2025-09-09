package logger

import "fmt"

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
)

type Log struct {
	err error
}

func New() *Log {
	return &Log{}
}

func (l *Log) WithError(err error) *Log {
	l.err = err
	return l
}

func (l *Log) Warn(s string) {
	if l.err != nil {
		fmt.Printf("❌:%s. err :%w", ColorYellow, s, l.err, ColorReset)
		return
	}
	fmt.Printf("❌:%s", ColorYellow, s, ColorReset)
}

func (l *Log) Error(s string) {
	if l.err != nil {
		fmt.Printf("❌:%s. err :%w", ColorRed, s, l.err, ColorReset)
		return
	}
	fmt.Printf("❌:%s", ColorRed, s, ColorReset)
}
