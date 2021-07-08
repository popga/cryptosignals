package logging

import (
	"cryptoapi/internal/helpers"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Logger struct {
	*logrus.Logger
}

func createLogFile() (*os.File, error) {
	fmt.Println(viper.GetString("base.logs.folder"))
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fileFormat := time.Now().Format("log-2006-01-02-15-04-05")
	if err := helpers.CreateDirIfNotExist(viper.GetString("base.logs.folder")); err != nil {
		return nil, err
	}
	logLocation := filepath.Join(cwd, viper.GetString("base.logs.folder"), fmt.Sprintf("%s.log", fileFormat))
	logFile, err := os.OpenFile(logLocation, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, err
}
func NewLogger() (*os.File, *Logger, error) {
	logFile, err := createLogFile()
	if err != nil {
		return nil, nil, err
	}
	baseLogger := logrus.New()
	logger := &Logger{baseLogger}
	//logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(io.MultiWriter(os.Stderr, logFile))
	logger.SetLevel(logrus.DebugLevel)
	return logFile, logger, nil
}
