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

// defined flags
var (
	levelFlag   logLevelFlag
	loadMapFlag = flag.Bool("loadmap", false, "loads map")
)

func init() {
	levelFlag.value = slog.LevelWarn
	flag.Var(&levelFlag, "level", "log level name")
}
