package log

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/fabric8-services/fabric8-wit/configuration"

	log "github.com/Sirupsen/logrus"
	"github.com/goadesign/goa"
)

const defaultPackageName = "github.com/fabric8-services/fabric8-wit/"

var (
	logger = &log.Logger{
		Out:       os.Stderr,
		Formatter: new(log.TextFormatter),
		Hooks:     make(log.LevelHooks),
		Level:     getDefaultLogLevel(),
	}
)

// InitializeLogger creates a default logger with the given ouput format and log level
func InitializeLogger(logJSON bool, lvl string) {
	logger = log.New()

	logLevel, err := log.ParseLevel(lvl)
	if err != nil {
		log.Warnf("unable to parse log level configuration error: %q", err)
		logLevel = log.ErrorLevel // reset to ERROR
	}
	log.SetLevel(logLevel)
	logger.Level = logLevel

	if logJSON {
		customFormatter := new(log.JSONFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"

		log.SetFormatter(customFormatter)
		customFormatter.DisableTimestamp = false
		logger.Formatter = customFormatter
	} else {
		customFormatter := new(log.TextFormatter)
		customFormatter.FullTimestamp = true
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		log.SetFormatter(customFormatter)
		logger.Formatter = customFormatter
	}

	logger.Out = os.Stdout
}

// NewCustomizedLogger creates a custom logger specifying the desired log level
// and the log format flag. Returns the logger object and the error.
func NewCustomizedLogger(level string, logJSON bool) (*log.Logger, error) {
	logger := log.New()

	lv, err := log.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.Level = lv

	if logJSON {
		customFormatter := new(log.JSONFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"

		log.SetFormatter(customFormatter)
		customFormatter.DisableTimestamp = false

		log.SetLevel(log.InfoLevel)
		logger.Level = lv
		logger.Formatter = customFormatter
	} else {
		customFormatter := new(log.TextFormatter)
		customFormatter.FullTimestamp = true
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		log.SetFormatter(customFormatter)

		log.SetLevel(log.DebugLevel)
		logger.Level = lv
		logger.Formatter = customFormatter
	}

	logger.Out = os.Stdout

	return logger, nil
}

// Logger returns the current logger object.
func Logger() *log.Logger {
	return logger
}

// IsDebug returns true if logger is set at DebugLevel.
// Useful if you need to do extra work that takes time to build the log statement
// that is not required as part of normal execution flow
func IsDebug() bool {
	return logger.Level >= log.DebugLevel
}

// Error logs an error message that might contain the following attributes: pid,
// request id if provided by the context, file location of the caller, line that
// called the log Error function and the function name. Moreover, we can use the
// parameter fields to add additional attributes to the output message. Likewise
// format and args are used to print a detailed message with the reasons of the
// error log.
func Error(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		file, line, pName, fName, err := extractCallerDetails()
		if err == nil {
			entry = entry.WithField("file", file).WithField("pkg", pName).WithField("line", line).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
			identityID, err := extractIdentityID(ctx)
			if err == nil {
				entry = entry.WithField("identity_id", identityID)
			}

			if req := goa.ContextRequest(ctx); req != nil {
				// Let's log some request details
				if len(req.Header) > 0 {
					headers := make(map[string]interface{}, len(req.Header))
					for k, v := range req.Header {
						// Hide sensitive information
						if k == "Authorization" || k == "Cookie" {
							headers[string(k)] = "*****"
						} else {
							headers[string(k)] = v
						}
					}
					entry = entry.WithField("req_headers", headers)
				}
				if len(req.Params) > 0 {
					entry = entry.WithField("req_params", req.Params)
				}
				if req.ContentLength > 0 {
					if mp, ok := req.Payload.(map[string]interface{}); ok && mp != nil {
						entry = entry.WithField("req_payload", mp)
					} else {
						js, err := json.Marshal(req.Payload)
						if err != nil {
							js = []byte("<invalid JSON>")
						}
						entry = entry.WithField("req_payload", string(js))
					}
				}
			}
		}

		if len(args) > 0 {
			entry.WithFields(fields).Errorf(format, args)
		} else {
			entry.WithFields(fields).Errorln(format)
		}
	}
}

// Warn logs a warning message that might contain the following attributes:
// request id if provided by the context, the file and the
// function name that invoked the Warn() function. In this function, we can use
// the parameter fields to add additional attributes to the output of this
// message. Likewise format and args are used to print a detailed message with
// the reasons of the warning log.
func Warn(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.WarnLevel {
		entry := log.NewEntry(logger)

		file, _, pName, fName, err := extractCallerDetails()
		if err == nil {
			entry = entry.WithField("file", file).WithField("pkg", pName).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
			identityID, err := extractIdentityID(ctx)
			if err == nil { // Otherwise we don't use the identityID
				entry = entry.WithField("identity_id", identityID)
			}
		}

		if len(args) > 0 {
			entry.WithFields(fields).Warnf(format, args...)
		} else {
			entry.WithFields(fields).Warnln(format)
		}
	}
}

// Info logs an info message that might contain the request id if provided by
// the context. In this function, the parameter fields enables to additional
// attributes to the message. The format and args input arguments are used to
// print a detailed information about the reasons of this log.
func Info(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.InfoLevel {
		entry := log.NewEntry(logger)

		_, _, pName, _, err := extractCallerDetails()
		if err == nil {
			entry = entry.WithField("pkg", pName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
			identityID, err := extractIdentityID(ctx)
			if err == nil { // Otherwise we don't use the identityID
				entry = entry.WithField("identity_id", identityID)
			}
		}

		if len(args) > 0 {
			entry.WithFields(fields).Infof(format, args...)
		} else {
			entry.WithFields(fields).Infoln(format)
		}
	}
}

// Panic logs a panic message that might contain the following attributes:
// the request id if provided by the context and the pid. In this function, the
// parameter fields enables to additional attributes to the message. The format
// and args input arguments are used to print a detailed information about the
// reasons of this log.
func Panic(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
			identityID, err := extractIdentityID(ctx)
			if err == nil { // Otherwise we don't use the identityID
				entry = entry.WithField("identity_id", identityID)
			}
		}

		if len(args) > 0 {
			entry.WithFields(fields).Panicf(format, args)
		} else {
			entry.WithFields(fields).Panicln(format)
		}
	}
}

// Debug logs a debug message that might specifies the request id if provided by
// the context. In this function, the parameter fields enables to additional
// attributes to the message. The format and args input arguments are used to
// print a detailed information about the reasons of this log.
func Debug(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.DebugLevel {
		entry := log.NewEntry(logger)

		_, _, pName, _, err := extractCallerDetails()
		if err == nil {
			entry = entry.WithField("pkg", pName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
			identityID, err := extractIdentityID(ctx)
			if err == nil {
				entry = entry.WithField("identity_id", identityID)
			}
		}

		if len(args) > 0 {
			entry.WithFields(fields).Debugf(format, args...)
		} else {
			entry.WithFields(fields).Debugln(format)
		}
	}
}

// extractCallerDetails gets information about the file, line and function that
// called a certain logging method such as Error, Info, Debug, Warn and Panic.
func extractCallerDetails() (file string, line int, pkg string, function string, err error) {
	if pc, file, line, ok := runtime.Caller(2); ok {
		fName := runtime.FuncForPC(pc).Name()

		parts := strings.Split(fName, ".")
		pl := len(parts)
		pName := ""

		if parts[pl-2][0] == '(' {
			pName = strings.Join(parts[0:pl-2], ".")
		} else {
			pName = strings.Join(parts[0:pl-1], ".")
		}

		pName = strings.Replace(pName, defaultPackageName, "", -1)

		return file, line, pName, fName, nil
	}

	return "", 0, "", "", errors.New("unable to extract the caller details")
}

// getDefaultLogLevel extracts the log level out of the ENV variable. It is used
// in tests and as default static initialization of the log. If the ENV variable
// is not set then the log level is Info.
func getDefaultLogLevel() log.Level {
	config, err := configuration.NewConfigurationData("")
	if err != nil {
		log.Errorf("error getting configuration data")
	}

	logLevel, err := log.ParseLevel(config.GetLogLevel())
	if err != nil {
		log.Warnf("unable to parse log level configuration error: %q", err)
		return log.InfoLevel // reset to INFO
	}
	return logLevel
}
