package ai

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LangchainLogger struct {
	*zap.Logger
}

func (l *LangchainLogger) Debugf(format string, v ...interface{}) {
	l.log(zap.DebugLevel, format, v...)
}

func (l *LangchainLogger) Errorf(format string, v ...interface{}) {
	l.log(zap.ErrorLevel, format, v...)
}

func (l *LangchainLogger) Infof(format string, v ...interface{}) {
	l.log(zap.InfoLevel, format, v...)
}

func (l *LangchainLogger) Warnf(format string, v ...interface{}) {
	l.log(zap.WarnLevel, format, v...)
}

func (l *LangchainLogger) log(Level zapcore.Level, format string, v ...interface{}) {
	if ce := l.Check(Level, fmt.Sprintf(format, v...)); ce != nil {
		ce.Write()
	}
}
