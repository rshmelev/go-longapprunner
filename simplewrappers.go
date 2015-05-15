package golongapprunner

import "time"

func PrepareSimpleRun(cmdline string) (*LongRun, chan *StreamData) {
	x := &LongRun{Args: StringToArgs(cmdline), StopChannel: make(chan struct{}, 1)}
	ch := x.GetOrCreateLogsChannel()
	return x, ch
}

func RunSimple(cmdline string) (*LongRun, chan *StreamData) {
	x, ch := PrepareSimpleRun(cmdline)
	go x.Start()
	return x, ch
}

func RunLimitedInTime(cmdline string, timeout time.Duration) (*LongRun, chan *StreamData) {
	x, ch := PrepareSimpleRun(cmdline)
	x.Timeout = timeout
	go x.Start()
	return x, ch
}

//----------------------------------------

// to avoid memory leak, you should always read from string channel to the end
func LogsChan_toStringChan(ch chan *StreamData, stdooutPrefix string, stderrPrefix string) chan string {
	res := make(chan string, 1)
	go func() {
		for x := range ch {
			if x.stderr {
				res <- stderrPrefix + x.data
			} else {
				res <- stdooutPrefix + x.data
			}
		}
		close(res)
	}()
	return res
}
