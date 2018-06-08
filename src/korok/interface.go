package korok

var globalLogger *Logger

func Info(msg string, v ...interface{}) {
	globalLogger.Info(6, msg, v...)
}

func Fatal(msg string, v ...interface{}) {
	globalLogger.Fatal(6, msg, v...)
}

func init() {
	globalLogger = NewMario()
}
