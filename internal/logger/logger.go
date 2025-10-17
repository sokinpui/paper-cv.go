package logger

import (
	"fmt"
	"log"
	"os"
)

// Init sets up the global logger to write to the specified file path.
// It returns the log file, which the caller is responsible for closing.
func Init(logFilePath string) (*os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	return logFile, nil
}
