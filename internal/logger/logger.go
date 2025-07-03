package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func NewLogger(level string) *Logger {
	logger := logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	
	// Set JSON formatter
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
	})
	
	return &Logger{Logger: logger}
}

func NewFileLogger(level, logFile string) (*Logger, error) {
	logger := NewLogger(level)
	
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}
	
	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	
	// Set output to both file and stdout
	logger.SetOutput(io.MultiWriter(os.Stdout, file))
	
	return logger, nil
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