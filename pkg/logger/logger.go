package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Interface -.
type Interface interface {
	Debug(message interface{}, args ...interface{})
	Info(message string, args ...interface{})
	Infof(format string, args ...interface{})
	Warn(message string, args ...interface{})
	Warnf(format string, args ...interface{})
	Error(message interface{}, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(message interface{}, args ...interface{})
}

// Logger -.
type Logger struct {
	logger zerolog.Logger
	infoW  io.Writer
	warnW  io.Writer
	errW   io.Writer
}

var _ Interface = (*Logger)(nil)

// New -.
func New(level string) *Logger {
	_ = os.MkdirAll("logger", os.ModePerm)

	infoFile, _ := os.OpenFile("logger/info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	warnFile, _ := os.OpenFile("logger/warning.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	errorFile, _ := os.OpenFile("logger/errors.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	var l zerolog.Level
	switch strings.ToLower(level) {
	case "error":
		l = zerolog.ErrorLevel
	case "warn":
		l = zerolog.WarnLevel
	case "info":
		l = zerolog.InfoLevel
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	return &Logger{
		logger: logger,
		infoW:  infoFile,
		warnW:  warnFile,
		errW:   errorFile,
	}
}

func (l *Logger) write(w io.Writer, level string, msg string) {
	line := fmt.Sprintf(`time=%s level=%s msg="%s"`+"\n",
		time.Now().UTC().Format(time.RFC3339Nano),
		level,
		msg,
	)
	w.Write([]byte(line))
}

// Debug -.
func (l *Logger) Debug(message interface{}, args ...interface{}) {
	l.logger.Debug().Msgf(fmt.Sprint(message), args...)
}

// Info -.
func (l *Logger) Info(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	l.logger.Info().Msg(msg)
	l.write(l.infoW, "INFO", msg)
}

// Infof -.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(format, args...)
}

// Warn -.
func (l *Logger) Warn(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	l.logger.Warn().Msg(msg)
	l.write(l.warnW, "WARN", msg)
}

// Warnf -.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(format, args...)
}

// Error -.
func (l *Logger) Error(message interface{}, args ...interface{}) {
	msg := fmt.Sprintf(fmt.Sprint(message), args...)
	l.logger.Error().Msg(msg)
	l.write(l.errW, "ERROR", msg)
}

// Errorf -.
func (l *Logger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Error().Msg(msg)
	l.write(l.errW, "ERROR", msg)
}

// Fatal -.
func (l *Logger) Fatal(message interface{}, args ...interface{}) {
	msg := fmt.Sprintf(fmt.Sprint(message), args...)
	l.logger.Fatal().Msg(msg)
	l.write(l.errW, "FATAL", msg)
	os.Exit(1)
}
