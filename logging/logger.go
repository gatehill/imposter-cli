package logging

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func SetLogLevel(lvl string) {
	// quieten the default logger
	logrus.SetLevel(logrus.InfoLevel)

	ll, err := logrus.ParseLevel(lvl)
	if err != nil {
		ll = logrus.DebugLevel
	}
	logger.SetLevel(ll)
}

// GetLogger returns the configured logger.
func GetLogger() *logrus.Logger {
	return logger
}
