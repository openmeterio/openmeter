package kafka

import (
	"fmt"
)

type LoggerFunc func(fmt string, args ...any)

type SaramaLoggerAdaptor struct {
	loggerFunc LoggerFunc
}

func (s *SaramaLoggerAdaptor) Print(v ...interface{}) {
	s.loggerFunc(fmt.Sprint(v...))
}

func (s *SaramaLoggerAdaptor) Printf(format string, v ...interface{}) {
	s.loggerFunc(fmt.Sprintf(format, v...))
}

func (s *SaramaLoggerAdaptor) Println(v ...interface{}) {
	s.loggerFunc(fmt.Sprint(v...))
}
