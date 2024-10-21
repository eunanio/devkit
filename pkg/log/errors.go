package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

func init() {
	path, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		homePath, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to get home directory")
			os.Exit(1)
		}

		path = fmt.Sprintf("%s/devkit.log", homePath)
	}

	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open log file")
		os.Exit(1)
	}

	if isDebug() {
		mw := io.MultiWriter(os.Stdout, logFile)
		logger := slog.New(slog.NewJSONHandler(mw, nil))
		slog.SetDefault(logger)
		return
	}

	logger := slog.New(slog.NewJSONHandler(logFile, nil))
	slog.SetDefault(logger)
}

func NoError(err error, msg string) {
	if err != nil {
		slog.Error(err.Error())
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

func isDebug() bool {
	_, ok := os.LookupEnv("DEBUG")
	return ok
}
