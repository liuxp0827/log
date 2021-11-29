package log

import (
	"io"
	"time"

	"github.com/evalphobia/logrus_sentry"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var defaultLogger *LogrusLogger

func GetLogger() *LogrusLogger {
	return defaultLogger
}

type LogrusLogger struct {
	Logger *logrus.Logger
}

func NewLogrusLogger(options ...LogrusOption) Logger {
	lg := logrus.New()
	lg.Level = logrus.DebugLevel

	lg.Formatter = &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05.999",
	}
	for _, option := range options {
		option(lg)
	}
	defaultLogger = &LogrusLogger{
		Logger: lg,
	}
	return defaultLogger
}

func (l *LogrusLogger) Log(level Level, keyvals ...interface{}) (err error) {
	var (
		logrusLevel logrus.Level
		fields      logrus.Fields = make(map[string]interface{})
		msg         string
	)

	switch level {
	case LevelDebug:
		logrusLevel = logrus.DebugLevel
	case LevelInfo:
		logrusLevel = logrus.InfoLevel
	case LevelWarn:
		logrusLevel = logrus.WarnLevel
	case LevelError:
		logrusLevel = logrus.ErrorLevel
	default:
		logrusLevel = logrus.DebugLevel
	}

	if logrusLevel > l.Logger.Level {
		return
	}

	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		if key == logrus.FieldKeyMsg {
			msg, _ = keyvals[i+1].(string)
			continue
		}
		if key == "stack" {
			msg, _ = keyvals[i+1].(string)
			continue
		}
		if len(key) == 0 { //FIXME kratos logger.Helper.WithContext 的bug
			msg, _ = keyvals[i+1].(string)
			continue
		}
		fields[key] = keyvals[i+1]
	}

	if len(fields) > 0 {
		l.Logger.WithFields(fields).Log(logrusLevel, msg)
	} else {
		l.Logger.Log(logrusLevel, msg)
	}

	return
}

type LogrusOption func(log *logrus.Logger)

func LevelOption(level string) LogrusOption {
	return func(log *logrus.Logger) {
		var err error
		log.Level, err = logrus.ParseLevel(level)
		if err != nil {
			log.Level = logrus.InfoLevel
		}
	}
}

func FsOption(path string, maxAge, maxSize int) LogrusOption {
	return func(log *logrus.Logger) {
		var writer io.Writer
		var err error

		writer, err = rotatelogs.New(
			path+".%Y%m%d",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour), // 保留最近 maxAge 天的日志
			rotatelogs.WithRotationSize(int64(maxSize*1024*1024)),     // 单日志超过 maxSize MB, 进行滚动
		)
		if err != nil {
			panic(err)
		}

		log.Hooks.Add(lfshook.NewHook(
			writer,
			&logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05.999",
			},
		))

	}
}

func SentryOption(dsn string) LogrusOption {
	return func(log *logrus.Logger) {
		hook, err := logrus_sentry.NewSentryHook(dsn, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		if err != nil {
			panic(err)
		}
		hook.StacktraceConfiguration.Enable = true
		log.Hooks.Add(hook)
	}
}

func OutputOption(w io.Writer) LogrusOption {
	return func(log *logrus.Logger) {
		log.Out = w
	}
}

func FormatterOption(formatter logrus.Formatter) LogrusOption {
	return func(log *logrus.Logger) {
		log.Formatter = formatter
	}
}

func OutputsOption(writers ...io.Writer) LogrusOption {
	return func(log *logrus.Logger) {
		log.Out = io.MultiWriter(writers...)
	}
}
