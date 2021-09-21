package engine

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"time"
)

const defaultStartTimeout = 30 * time.Second

func getStartTimeout() time.Duration {
	startTimeout := viper.GetInt("startTimeout")
	if startTimeout == 0 {
		return defaultStartTimeout
	}
	return time.Duration(startTimeout) * time.Second
}

func WaitUntilUp(port int) {
	url := fmt.Sprintf("http://localhost:%d/system/status", port)
	logrus.Tracef("waiting for mock engine to come up at %v", url)

	startedC := make(chan bool)
	max := time.NewTimer(getStartTimeout())
	defer max.Stop()

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			resp, err := http.Get(url)
			if err != nil {
				continue
			}
			if _, err := io.ReadAll(resp.Body); err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				startedC <- true
				break
			}
		}
	}()

	select {
	case <-max.C:
		logrus.Fatal("timed out waiting for engine to start")
		break
	case <-startedC:
		logrus.Tracef("engine started")
	}
}
