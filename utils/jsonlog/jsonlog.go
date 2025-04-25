package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Level int8

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger 定义一个日志记录器
// 用于记录应用程序的日志信息，包括日志级别、消息、元数据等
// 可以用于记录应用程序的运行状态、错误信息、调试信息等
// 可以用于记录应用程序的运行状态、错误信息、调试信息等
type Logger struct {
	out      io.Writer
	minLevel Level
	pretty   bool
	mu       sync.Mutex
}

func New(out io.Writer, minLevel Level, pretty bool) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
		pretty:   pretty,
	}
}

// PrintInfo 打印信息
// 用于打印信息，包括日志级别、消息、元数据等
// 可以用于打印应用程序的运行状态、错误信息、调试信息等
// 可以用于打印应用程序的运行状态、错误信息、调试信息等
func (l *Logger) Info(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

func (l *Logger) Error(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

func (l *Logger) Fatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
	os.Exit(1)
}

func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	if level < l.minLevel {
		return 0, nil
	}

	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	var line []byte
	var err error
	if l.pretty {
		line, err = json.MarshalIndent(aux, "", "  ")
		if err != nil {
			line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
		}
		s := string(line)
		s = strings.ReplaceAll(s, "\\n", "\n    ")
		s = strings.ReplaceAll(s, "\\t", "  ")
		line = []byte(s)
	} else {
		line, err = json.Marshal(aux)
		if err != nil {
			line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.out.Write(append(line, '\n'))
}
