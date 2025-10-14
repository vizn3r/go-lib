package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// ANSI colors
type Color string

const (
	Reset   Color = "\033[0m"
	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Purple  Color = "\033[35m"
	Magenta Color = "\033[35m"
	Grey    Color = "\033[90m"
	Cyan    Color = "\033[36m"
)

// Highlight keywords
var highlights = map[string]Color{
	"OK":    Green,
	"ERROR": Red,
	"FAIL":  Red,

	// HTTP Methods
	"GET":     Blue,
	"POST":    Cyan,
	"PUT":     Yellow,
	"DELETE":  Purple,
	"PATCH":   Magenta,
	"OPTIONS": Cyan,
	"HEAD":    Blue,
}

type LogLevel int

// Log message struct for channel
type logMessage struct {
	level LogLevel
	msg   string
}

// Logger wraps log.Logger and a channel for async logging
type Logger struct {
	l      *log.Logger
	logCh  chan logMessage
	done   chan struct{}
	closed bool

	printTime   bool
	maxLogLevel LogLevel
	sync        bool

	color  Color
	module string
}

const (
	LevelDisabled LogLevel = iota - 1
	LevelPrint
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// New creates a new async Logger for a module
func New(module string, color Color, writers ...io.Writer) *Logger {
	levelStr := os.Getenv("LOGGER_LEVEL")
	level := LevelPrint
	if levelStr != "" {
		switch levelStr {
		case "disabled", "none", "off":
			level = LevelDisabled
		case "debug":
			level = LevelDebug
		case "info":
			level = LevelInfo
		case "warn":
			level = LevelWarn
		case "error":
			level = LevelError
		case "fatal":
			level = LevelFatal
		case "all":
			level = LevelPrint
		}
	}

	out := io.MultiWriter(os.Stdout)
	if len(writers) > 0 {
		out = io.MultiWriter(writers...)
	}

	prefix := fmt.Sprintf("%s[%s]%s ", color, module, Grey)
	lg := &Logger{
		l:           log.New(out, prefix, log.LstdFlags),
		logCh:       make(chan logMessage, 100), // buffered channel
		done:        make(chan struct{}),
		maxLogLevel: level,
		printTime:   true,
		color:       color,
		module:      module,
	}

	// start logger goroutine
	go lg.run()

	return lg
}

func (lg *Logger) SetLevel(level LogLevel) {
	lg.maxLogLevel = level
}

func (lg *Logger) SetSync(sync bool) {
	lg.sync = sync
}

func (lg *Logger) SetPrintTime(print bool) {
	lg.printTime = print
	if !print {
		lg.l.SetFlags(0)
	} else {
		prefix := fmt.Sprintf("%s[%s]%s ", lg.color, lg.module, Grey)
		lg.l.SetPrefix(prefix)
		lg.l.SetFlags(log.LstdFlags)
	}
}

// run listens on the channel and prints messages
func (lg *Logger) run() {
	for m := range lg.logCh {
		if m.level < lg.maxLogLevel {
			continue
		}
		switch m.level {
		case LevelInfo:
			lg.l.Printf(fmt.Sprintf("%s[INFO]%s %s", Blue, Reset, m.msg))
		case LevelWarn:
			lg.l.Printf(fmt.Sprintf("%s[WARN]%s %s", Yellow, Reset, m.msg))
		case LevelError:
			lg.l.Printf(fmt.Sprintf("%s[ERROR]%s %s", Red, Reset, m.msg))
		case LevelDebug:
			lg.l.Printf(fmt.Sprintf("%s[DEBUG]%s %s", Grey, Reset, m.msg))
		case LevelFatal:
			lg.l.Printf(fmt.Sprintf("%s[FATAL]%s %s", Red, Reset, m.msg))
			os.Exit(1)
		case LevelPrint:
			lg.l.Printf("%s%s", Reset, colorString(m.msg))
		default:
			lg.l.Printf("%s%s", Reset, m.msg)
		}
	}
	close(lg.done)
}

func (lg *Logger) printer(m logMessage) {
	switch m.level {
	case LevelInfo:
		lg.l.Printf(fmt.Sprintf("%s[INFO]%s %s", Blue, Reset, m.msg))
	case LevelWarn:
		lg.l.Printf(fmt.Sprintf("%s[WARN]%s %s", Yellow, Reset, m.msg))
	case LevelError:
		lg.l.Printf(fmt.Sprintf("%s[ERROR]%s %s", Red, Reset, m.msg))
	case LevelDebug:
		lg.l.Printf(fmt.Sprintf("%s[DEBUG]%s %s", Grey, Reset, m.msg))
	case LevelFatal:
		lg.l.Printf(fmt.Sprintf("%s[FATAL]%s %s", Red, Reset, m.msg))
		os.Exit(1)
	case LevelPrint:
		lg.l.Printf("%s%s", Reset, colorString(m.msg))
	default:
		lg.l.Printf("%s%s", Reset, m.msg)
	}
}

// Log pushes a message to the log channel
func (lg *Logger) Log(level LogLevel, v ...any) {
	m := logMessage{level: level, msg: fmt.Sprint(v...)}
	if lg.sync {
		if level < lg.maxLogLevel {
			return
		}
		lg.printer(m)
		return
	}
	lg.logCh <- m
}

// Info pushes a message to the log channel
func (lg *Logger) Info(v ...any) {
	lg.Log(LevelInfo, v...)
}

// Warn pushes a message to the log channel
func (lg *Logger) Warn(v ...any) {
	lg.Log(LevelWarn, v...)
}

// Error pushes a message to the log channel
func (lg *Logger) Error(v ...any) {
	lg.Log(LevelError, v...)
}

func (lg *Logger) Debug(v ...any) {
	lg.Log(LevelDebug, v...)
}

// Print pushes a colored message to the log channel
func (lg *Logger) Print(v ...any) {
	lg.Log(LevelPrint, v...)
}

// Fatal pushes a message to the log channel and exits
func (lg *Logger) Fatal(v ...any) {
	lg.Log(LevelFatal, v...)
}

func Hyperlink(url string, v ...any) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, fmt.Sprint(v...))
}

// Close the logger (flushes remaining messages)
func (lg *Logger) Close() {
	if !lg.closed {
		close(lg.logCh)
		<-lg.done
		lg.closed = true
	}
}

// colorString replaces keywords with colored versions
func colorString(s string) string {
	for word, color := range highlights {
		s = strings.ReplaceAll(s, word, fmt.Sprintf("%s%s%s", color, word, Reset))
	}
	return s
}
