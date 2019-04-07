package log

var (
	DefaultLogger = func(tmpl string, s ...interface{}) {}
)

// null logger
type loggerType func(tmpl string, s ...interface{})

func SetLogger(logger loggerType) {
	DefaultLogger = logger
}
