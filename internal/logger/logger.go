package logger

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// CleanFormatter provides clean output with just the message
type CleanFormatter struct{}

func (f *CleanFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

type Logger struct {
	*logrus.Logger
}

func NewLogger(level string) *Logger {
	return NewLoggerWithFormat(level, "clean")
}

func NewLoggerWithFormat(level, format string) *Logger {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter based on format
	switch strings.ToLower(format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
		})
	case "clean":
		logger.SetFormatter(&CleanFormatter{})
	default: // "text" or any other value defaults to text
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
			DisableColors:   false,
			FullTimestamp:   true,
		})
	}

	return &Logger{Logger: logger}
}

func NewFileLogger(level, logFile string) (*Logger, error) {
	return NewFileLoggerWithFormat(level, logFile, "clean")
}

func NewFileLoggerWithFormat(level, logFile, format string) (*Logger, error) {
	return NewFileLoggerWithSeparateFormats(level, logFile, format, format)
}

func NewFileLoggerWithSeparateFormats(level, logFile, stdoutFormat, fileFormat string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Create main logger for stdout with clean format
	logger := NewLoggerWithFormat(level, stdoutFormat)
	logger.SetOutput(os.Stdout)

	// Add hook for file logging with different format
	fileHook := &FileHook{
		file:       file,
		fileFormat: fileFormat,
	}
	logger.AddHook(fileHook)

	return logger, nil
}

// FileHook implements logrus.Hook interface for file logging with different format
type FileHook struct {
	file       *os.File
	fileFormat string
}

func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *FileHook) Fire(entry *logrus.Entry) error {
	// Create a temporary logger with file format
	tempLogger := logrus.New()
	tempLogger.SetOutput(hook.file)

	// Set formatter based on file format
	switch strings.ToLower(hook.fileFormat) {
	case "json":
		tempLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
		})
	case "clean":
		tempLogger.SetFormatter(&CleanFormatter{})
	default: // "text"
		tempLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
			DisableColors:   false,
			FullTimestamp:   true,
		})
	}

	// Log to file with the appropriate format
	tempLogger.WithFields(entry.Data).Log(entry.Level, entry.Message)

	return nil
}

func (l *Logger) WithDatabase(dbName string) *logrus.Entry {
	return l.WithField("database", dbName)
}

func (l *Logger) WithError(err error) *logrus.Entry {
	return l.WithField("error", err.Error())
}

func (l *Logger) WithBackupFile(fileName string) *logrus.Entry {
	return l.WithField("backup_file", fileName)
}
