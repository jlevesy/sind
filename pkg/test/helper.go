package test

import "time"

func retry(retries int, period time.Duration, cb func() error) error {
	var err error

	for retries > 0 {
		err = cb()
		if err == nil {
			return nil
		}

		<-time.After(period)
		retries--
	}

	return err
}
