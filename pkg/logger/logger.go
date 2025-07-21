package logger

import (
	"fmt"
	"os"
	"time"
)

// LogLevel определяет уровни логирования с использованием битовых флагов.
type LogLevel int

const (
	None         LogLevel = 0x00000000 // Ничего не логировать
	InfoLevel    LogLevel = 0x00000001 // Информационные сообщения
	ErrorLevel   LogLevel = 0x00000010 // Сообщения об ошибках
	DebugInfo    LogLevel = 0x00000100 // Отладочная информация
	WarningLevel LogLevel = 0x00001000 // Предупреждающие сообщения
	FatalLevel   LogLevel = 0x00010000 // Критические ошибки, приводящие к выходу из программы

	AllLevels LogLevel = InfoLevel | ErrorLevel | DebugInfo | WarningLevel | FatalLevel // Все уровни
)

// Logger определяет интерфейс для системы логирования.
type Logger interface {
	SetLogLevel(level LogLevel)
	Log(level LogLevel, format string, args ...interface{})
	Info(format string, args ...interface{})
	DebugInfo(format string, args ...interface{})
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})  // Добавлен уровень предупреждений
	Fatal(format string, args ...interface{}) // Добавлен критический уровень
}

// ConsoleLogger является реализацией Logger, которая выводит сообщения в консоль.
type ConsoleLogger struct {
	currentLogLevel LogLevel
}

// NewConsoleLogger создает новый экземпляр ConsoleLogger с заданным начальным уровнем.
func NewConsoleLogger(initialLogLevel LogLevel) *ConsoleLogger {
	return &ConsoleLogger{
		currentLogLevel: initialLogLevel,
	}
}

// SetLogLevel устанавливает текущий уровень логирования.
func (l *ConsoleLogger) SetLogLevel(level LogLevel) {
	l.currentLogLevel = level
}

// Log выводит сообщение с заданным уровнем.
func (l *ConsoleLogger) Log(level LogLevel, format string, args ...interface{}) {
	if (l.currentLogLevel & level) != 0 {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logMessage := fmt.Sprintf(format, args...)
		fmt.Printf("[%s][%s] %s\n", timestamp, l.levelToString(level), logMessage)

		if level == FatalLevel {
			os.Exit(1) // При фатальной ошибке завершаем выполнение программы
		}
	}
}

// Info логирует информационное сообщение.
func (l *ConsoleLogger) Info(format string, args ...interface{}) {
	l.Log(InfoLevel, format, args...)
}

// DebugInfo логирует отладочную информацию.
func (l *ConsoleLogger) DebugInfo(format string, args ...interface{}) {
	l.Log(DebugInfo, format, args...)
}

// Error логирует сообщение об ошибке.
func (l *ConsoleLogger) Error(format string, args ...interface{}) {
	l.Log(ErrorLevel, format, args...)
}

// Warn логирует предупреждающее сообщение.
func (l *ConsoleLogger) Warn(format string, args ...interface{}) {
	l.Log(WarningLevel, format, args...)
}

// Fatal логирует критическую ошибку и завершает программу.
func (l *ConsoleLogger) Fatal(format string, args ...interface{}) {
	l.Log(FatalLevel, format, args...)
}

// levelToString Helper для преобразования LogLevel в строку.
func (l *ConsoleLogger) levelToString(level LogLevel) string {
	switch level {
	case InfoLevel:
		return "INFO"
	case ErrorLevel:
		return "ERROR"
	case DebugInfo:
		return "DEBUG"
	case WarningLevel:
		return "WARNING"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
