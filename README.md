# go-longapprunner

It is probably the most powerful and simple way to run external app and monitor its logs and its activity.

1. Pass simple command line as you pass it to console
2. Both stderr and stdout are returned in single chan
3. Works with tools like ffmpeg 
4. Attempts to kill processes using external tools which makes it possible to kill process trees
5. Supports [github.com/rshmelev/go-inthandler]
6. Supports cancelling the run using the StopChannel
7. Support for timeouts 


```
r, ch := RunSimple("ping google.com")

// channel is closed when both stderr and stdout are closed
for x := range LogsChan_toStringChan(ch, "> ", "! ") {
	println(x)
}
```

After process is finished, you can check this:

 - StoppedItself - wasn't stopped by the app
 - TimeoutHappened (use `RunLimitedTime()` func or `PrepareSimpleRun()` which allow to set Timeout)
 - Summary() is cool for debugging, will output something like this: `Process [ping google.com] (pid 14680) was running 3.37s (systime:0.28s, usertime:0.00s) and exited, stopped itself, logs amount is 436 bytes`
 - ForcefullyKilled 
 - ExitError
 - Duration
 - BytesLogged

During the app run, you can use this:

- WaitForFinish_ofLogProcessing() - actually this means ability to catch real app stop

Setting up long-app-runner with `PrepareSimpleRun()` includes ability to set the following options before starting it with `go r.Start()`:

- Timeout - no timeout by default
- TimeToWaitForCleanup - by default it is 1 second
- SplitFunc - by default, own func ModScanLines is used, you can use bufio.ScanLines instead, however it will not work well with tools like `ffmpeg` which are using \r instead of \r?\n
- ShutdownURL - ability to send special HTTP request to app instead killing it with some brutal way
- StopChannel

author: rshmelev@gmail.com