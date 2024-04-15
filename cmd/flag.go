package main

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

type logLevelFlag struct {
	value slog.Level
}

func (l *logLevelFlag) String() string {
	return l.value.String()
}

func (l *logLevelFlag) Set(value string) error {
	m := map[string]slog.Level{"DEBUG": slog.LevelDebug, "INFO": slog.LevelInfo, "WARN": slog.LevelWarn, "ERROR": slog.LevelError}
	v, ok := m[strings.ToUpper(value)]
	if !ok {
		return fmt.Errorf("unknown log level")
	}
	l.value = v
	return nil
}

var levelFlag logLevelFlag
var createDBFlag = flag.Bool("createdb", false, "whether to create the database")

func init() {
	levelFlag.value = slog.LevelInfo
	flag.Var(&levelFlag, "level", "log level name")
}
