package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
)

// ANSI colors
type Color string

const (
	Reset Color = "\033[0m"

	// Regular colors
	Black   Color = "\033[30m"
	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Magenta Color = "\033[35m"
	Cyan    Color = "\033[36m"
	White   Color = "\033[37m"
	Grey    Color = "\033[90m"

	// Bright colors
	BrightRed     Color = "\033[91m"
	BrightGreen   Color = "\033[92m"
	BrightYellow  Color = "\033[93m"
	BrightBlue    Color = "\033[94m"
	BrightMagenta Color = "\033[95m"
	BrightCyan    Color = "\033[96m"
	BrightWhite   Color = "\033[97m"
)

// Highlighter pairs a regex pattern with a color
type Highlighter struct {
	Pattern *regexp.Regexp
	Color   Color
}

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
	colorOutput bool

	color  Color
	module string
}

type LogLevel int

const (
	LevelDisabled LogLevel = iota - 1
	LevelPrint
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var loggers map[string]*Logger

var highlighters []Highlighter

// Default highlights
func init() {
	// Keywords
	keywords := map[string]Color{
		"OK":    Green,
		"ERROR": Red,
		"FAIL":  Red,

		// HTTP Methods
		"GET":     Green,
		"POST":    Blue,
		"PUT":     Yellow,
		"DELETE":  Cyan,
		"PATCH":   Magenta,
		"OPTIONS": Cyan,
		"HEAD":    Blue,

		// Errors
		"error": Red,
		"Error": Red,
	}

	for word, color := range keywords {
		highlighters = append(highlighters, Highlighter{
			Pattern: regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`),
			Color:   color,
		})
	}

	// Strings (double-quoted and single-quoted)
	AddHighlighter(`"[^"]*"|'[^']*'`, Yellow)

	// Numbers (integers, floats, hex)
	AddHighlighter(`0x[0-9A-Fa-f]+|\b\d+\.\d+|\b\d+\b`, Cyan)
}

// AddHighlighter adds a regex pattern with a color to the highlighter list
func AddHighlighter(pattern string, color Color) {
	highlighters = append(highlighters, Highlighter{
		Pattern: regexp.MustCompile(pattern),
		Color:   color,
	})
}

// New creates a new async logger
func New(module string, color Color, writers ...io.Writer) *Logger {
	// Defaults
	sync := false
	printTime := true
	level := LevelPrint
	colorOutput := true
	fast := false
	global := true

	levelStr := os.Getenv("LOGGER_LEVEL")
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

	sync, err := strconv.ParseBool(os.Getenv("LOGGER_SYNC"))
	if err != nil {
		sync = false
	}
	printTime, err = strconv.ParseBool(os.Getenv("LOGGER_TIME"))
	if err != nil {
		printTime = true
	}
	colorOutput, err = strconv.ParseBool(os.Getenv("LOGGER_COLORS"))
	if err != nil {
		colorOutput = true
	}
	fast, err = strconv.ParseBool(os.Getenv("LOGGER_FAST"))
	if err != nil {
		fast = false
	}
	global, err = strconv.ParseBool(os.Getenv("LOGGER_GLOBAL"))
	if err != nil {
		global = false
	}

	if fast {
		level = LevelPrint
		sync = true
		printTime = false
		colorOutput = false
		global = false
	}

	if global {
		loggers = make(map[string]*Logger)
	}

	out := io.MultiWriter(os.Stdout)
	if len(writers) > 0 {
		out = io.MultiWriter(writers...)
	}

	prefix := fmt.Sprintf("%s[%s]%s ", color, module, Grey)
	if !colorOutput {
		prefix = fmt.Sprintf("[%s] ", module)
	}
	lg := &Logger{
		l:           log.New(out, prefix, log.LstdFlags),
		logCh:       make(chan logMessage, 100), // buffered channel
		done:        make(chan struct{}),
		maxLogLevel: level,
		printTime:   printTime,
		sync:        sync,
		colorOutput: colorOutput,
		color:       color,
		module:      module,
	}

	if global {
		loggers[module] = lg
	}

	// start logger goroutine
	if !sync {
		go lg.run()
	} else {
		lg.closed = true
	}

	if !lg.printTime {
		lg.l.SetFlags(0)
	} else {
		prefix := fmt.Sprintf("%s[%s]%s ", lg.color, lg.module, Grey)
		lg.l.SetPrefix(prefix)
		lg.l.SetFlags(log.LstdFlags)
	}

	return lg
}

func GetLogger(module string) *Logger {
	return loggers[module]
}

func CloseAll() {
	for _, lg := range loggers {
		lg.Close()
	}
}

func (lg *Logger) SetLevel(level LogLevel) {
	lg.maxLogLevel = level
}

func (lg *Logger) SetSync(sync bool) {
	lg.sync = sync
}

func AddHighlight(word string, color Color) {
	AddHighlighter(`\b`+regexp.QuoteMeta(word)+`\b`, color)
}

func (lg *Logger) SetPrintTime(print bool) {
	lg.printTime = print
}

// run listens on the channel and prints messages
func (lg *Logger) run() {
	for m := range lg.logCh {
		if m.level < lg.maxLogLevel {
			continue
		}
		lg.printer(m)
	}
	close(lg.done)
}

func (lg *Logger) printer(m logMessage) {
	if m.level < lg.maxLogLevel {
		return
	}

	msg := m.msg
	if lg.colorOutput {
		msg = colorString(msg) // color the content
	}

	switch m.level {
	case LevelInfo:
		if lg.colorOutput {
			lg.l.Printf("%s[I]%s   %s", Blue, Reset, msg)
		} else {
			lg.l.Printf("[I]   %s", msg)
		}
	case LevelWarn:
		if lg.colorOutput {
			lg.l.Printf("%s[W] ? %s%s", Yellow, Reset, msg)
		} else {
			lg.l.Printf("[W] ? %s", msg)
		}
	case LevelError:
		if lg.colorOutput {
			lg.l.Printf("%s<E> ! %s%s", Red, Reset, msg)
		} else {
			lg.l.Printf("<E> ! %s", msg)
		}
	case LevelDebug:
		if lg.colorOutput {
			lg.l.Printf("%s[D]%s   %s", Grey, Reset, msg)
		} else {
			lg.l.Printf("[D]   %s", msg)
		}
	case LevelFatal:
		if lg.colorOutput {
			lg.l.Printf("%s<F>!!! %s%s", Red, Reset, msg)
		} else {
			lg.l.Printf("<F>!!! %s", msg)
		}
		os.Exit(1)
	default:
		lg.l.Printf("%s", msg)
	}
}

var msgPool = sync.Pool{
	New: func() any {
		return new(logMessage)
	},
}

// Log pushes a message to the log channel
func (lg *Logger) Log(level LogLevel, v ...any) {
	if level < lg.maxLogLevel {
		return
	}
	m := msgPool.Get().(*logMessage)
	m.level = level
	m.msg = colorString(fmt.Sprint(v...))

	if !lg.colorOutput {
		m.msg = fmt.Sprint(v...)
	}
	if lg.sync {
		lg.printer(*m)
	} else {
		lg.logCh <- *m
	}
	msgPool.Put(m)
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

func ColorString(c Color, s ...any) string {
	return fmt.Sprintf("%s%s%s", c, fmt.Sprint(s...), Reset)
}

// colorString applies all highlighters in order
func colorString(s string) string {
	for _, h := range highlighters {
		s = h.Pattern.ReplaceAllStringFunc(s, func(match string) string {
			return string(h.Color) + match + string(Reset)
		})
	}
	return s
}
