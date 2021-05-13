package log

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync"

	rotate "github.com/natefinch/lumberjack"
	"go.uber.org/zap/zapcore"
)

// Level defines Log Level
type Level int8

const (
	// DebugLevel logs produce a large size of logs so should be avoided in Production
	// in Development should be fine to use
	DebugLevel Level = iota
	// InfoLevel for Production environment
	InfoLevel
	// WarnLevel sth happened or received out of expectation
	WarnLevel
	// ErrorLevel for high-priority cases but program can still run
	ErrorLevel
	// PanicLevel logs a message, then panics
	PanicLevel
	// FatalLevel for crucial cases means program is not able run any more
	// after logging message then calls os.Exit(1)
	FatalLevel
)

var (
	mu sync.Mutex
	// Note that we have a dependency on the format of "ERROR" messages in the log
	// in the integration tests. So, if we change the logger here, the corresponding
	// entry in the integration tests needs to be changed.
	logger       = newZapLogger(os.Stdout)
	level  Level = InfoLevel

	levelMap = map[string]Level{
		"debug": DebugLevel,
		"info":  InfoLevel,
		"warn":  WarnLevel,
		"error": ErrorLevel,
		"fatal": FatalLevel,
	}

	// debugMap stores the set of messages that are suppressed.
	// TODO: add ttl support so it doesn't grow infinitely.
	debugMap = &sync.Map{}

	traceFiles = make(map[string]bool)

	dumpGoroutinesOnFatal = true
)

func init() {
	if files := os.Getenv("GS_TRACE_FILES"); files != "" {
		traceSet := strings.Split(files, ",")
		for i := range traceSet {
			traceFiles[strings.TrimSpace(traceSet[i])] = true
		}
	}
}

// SetLogFilePath is used for overriding default stdout writer
func SetLogFilePath(path string) {
	mu.Lock()
	defer mu.Unlock()
	rotator := &rotate.Logger{
		Filename: path,
		// Limits max size for any log file
		MaxSize: 50, // megabytes
		// Limits the total number of files can be rotated
		MaxBackups: 7,
		// Limits the age of any file in disk.
		MaxAge: 7, //days
	}

	// If the log file exists, rotates it once for the ease of use.
	if _, err := os.Stat(path); err == nil {
		_ = rotator.Rotate()
	}

	logger = newZapLogger(rotator)
}

// SetLevel to change the log level in run-time
func SetLevel(l Level) {
	mu.Lock()
	defer mu.Unlock()
	level = l
}

// TraceOn returns true if log current level is equal or lower than Debug and ED_TRACE_FILES environment variable is set. e.g. "*" or "pusher,notifier"
// currently min log level is Debug. In some conditions, creating trace string is also an expensive procedure, this flag can be used to eliminate such evaluation.
// if log.TraceOn() {
//     message := expensiveString()
//     log.Trace("My message %s", message)
// }
func TraceOn() bool {
	return level <= DebugLevel && len(traceFiles) > 0
}

// SetLevelWithName to change the log level with level name in run-time
// it works case-insensitive for levelName, if levelName is invalid
// then level is assigned to InfoLevel
func SetLevelWithName(levelName string) {
	mu.Lock()
	defer mu.Unlock()
	l, ok := levelMap[strings.ToLower(levelName)]
	if !ok {
		level = InfoLevel
	} else {
		level = l
	}
}

// DisableGoroutineDumpOnFatal disables the goroutine dump on fatal logs.
// It's not needed to print goroutines in some scenarios such as intentionally exiting the process in tools.
func DisableGoroutineDumpOnFatal() {
	mu.Lock()
	defer mu.Unlock()
	dumpGoroutinesOnFatal = false
}

func message(format string, args ...interface{}) string {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return msg
}

// Debug logs a message at DebugLevel. First parameter is a string to use
// formatting the message and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Debug(format string, args ...interface{}) {
	if level <= DebugLevel {
		logger.Debug(message(format, args...))
	}
}

// Trace logs a message at DebugLevel if ED_TRACE_FILES environment variable is set e.g. "*" or "pusher,notifier"
// First parameter is a string to use formatting the message and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Trace(format string, args ...interface{}) {
	if level <= DebugLevel && len(traceFiles) > 0 {
		msg := message(format, args...)
		if traceFiles["*"] {
			logger.Debug(msg)
			return
		}
		ce := logger.Check(zapcore.DebugLevel, msg)
		fileName := strings.Split(ce.Caller.TrimmedPath(), ":")[0]
		for name := range traceFiles {
			if strings.Contains(fileName, name) {
				logger.Debug(msg)
				return
			}
		}
	}
}

// DebugOnce logs the message at DebugLevel and does it only once.
// Useful for suppressing logs from noisy sources which periodically prints same thing over and over.
// Don't call DebugOnce for logs which vary in text e.g. log.DebugOnce("%s", time.Now()).
// The formatted message hash is kept in memory for future to check whether it was logged before.
func DebugOnce(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if _, loaded := debugMap.LoadOrStore(message, true); !loaded {
		if level <= DebugLevel {
			logger.Debug(message)
		}
	}
}

// Info logs a message at InfoLevel. First parameter is a string to use
// formatting the message and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Info(format string, args ...interface{}) {
	if level <= InfoLevel {
		logger.Info(message(format, args...))
	}
}

// Warn logs a message at WarnLevel. First parameter is a string to use
// formatting the message and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Warn(format string, args ...interface{}) {
	if level <= WarnLevel {
		logger.Warn(message(format, args...))
	}
}

// Error logs a message at ErrorLevel. First parameter is a string to use
// formatting the message and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Error(format string, args ...interface{}) {
	if level <= ErrorLevel {
		logger.Error(message(format, args...))
	}
}

// Panic logs a message at DebugLevel then the logger then panics.
// First parameter is a string to use formatting the message
// and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Panic(format string, args ...interface{}) {
	if level <= PanicLevel {
		logger.Panic(message(format, args...))
	}
}

// Fatal logs a message at FatalLevel then calls os.Exit(1)
// First parameter is a string to use formatting the message
// and it takes any number of arguments to build the log message.
// If you have just a log message without format then
// use just one parameter to build log message and logs it
func Fatal(format string, args ...interface{}) {
	if level <= FatalLevel {
		if dumpGoroutinesOnFatal {
			logger.Info(message("running goroutines: %s", runningGoRoutines()))
		}
		logger.Fatal(message(format, args...))
	}
}

// DebugC implements same functionality as Info with context
func DebugC(ctx context.Context, format string, args ...interface{}) {
	if level <= DebugLevel {
		fields := transformFunc(ctx)
		logger.Debug(message(format, args...), toZap(fields)...)
	}
}

// TraceC implements same functionality as Info with context
func TraceC(ctx context.Context, format string, args ...interface{}) {
	if level <= DebugLevel && len(traceFiles) > 0 {
		msg := message(format, args...)
		if traceFiles["*"] {
			logger.Debug(msg, toZap(transformFunc(ctx))...)
			return
		}
		ce := logger.Check(zapcore.DebugLevel, msg)
		fileName := strings.Split(ce.Caller.TrimmedPath(), ":")[0]
		for name := range traceFiles {
			if strings.Contains(fileName, name) {
				logger.Debug(msg, toZap(transformFunc(ctx))...)
				return
			}
		}
	}
}

// InfoC implements same functionality as Info with context
func InfoC(ctx context.Context, format string, args ...interface{}) {
	if level <= InfoLevel {
		fields := transformFunc(ctx)
		logger.Info(message(format, args...), toZap(fields)...)
	}
}

// WarnC implements same functionality as Info with context
func WarnC(ctx context.Context, format string, args ...interface{}) {
	if level <= WarnLevel {
		fields := transformFunc(ctx)
		logger.Warn(message(format, args...), toZap(fields)...)
	}
}

// ErrorC implements same functionality as Info with context
func ErrorC(ctx context.Context, format string, args ...interface{}) {
	if level <= ErrorLevel {
		fields := transformFunc(ctx)
		logger.Error(message(format, args...), toZap(fields)...)
	}
}

// PanicC implements same functionality as Info with context
func PanicC(ctx context.Context, format string, args ...interface{}) {
	if level <= PanicLevel {
		fields := transformFunc(ctx)
		logger.Panic(message(format, args...), toZap(fields)...)
	}
}

// FatalC implements same functionality as Info with context
func FatalC(ctx context.Context, format string, args ...interface{}) {
	if level <= FatalLevel {
		fields := transformFunc(ctx)
		if dumpGoroutinesOnFatal {
			logger.Info(message("running goroutines: %s", runningGoRoutines()), toZap(fields)...)
		}
		logger.Fatal(message(format, args...), toZap(fields)...)
	}
}

func runningGoRoutines() string {
	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, 1)
	return buf.String()
}
