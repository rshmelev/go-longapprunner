package golongapprunner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	interrupts "github.com/rshmelev/go-inthandler"
	. "github.com/rshmelev/go-ternary/if"
)

// packets got from stdout/err of external process
type StreamData struct {
	Stderr bool
	// if data is nil then certain pipe is closed
	Data string
}

type LongRun struct {
	// main
	Args        []string
	LogsChannel chan *StreamData
	WorkingDir  string

	// config
	StopChannel          chan struct{}
	Timeout              time.Duration
	TimeToWaitForCleanup time.Duration
	SplitFunc            bufio.SplitFunc
	ShutdownURL          string // idea is to send HTTP request to stop something

	// vars
	Cmd                      *exec.Cmd
	stdo                     io.ReadCloser
	stde                     io.ReadCloser
	LogsProcessingIsOverChan chan struct{}

	// result
	ExitError    error
	StartTime    time.Time
	Duration     time.Duration
	ProcessState *os.ProcessState
	// some more details
	StoppedItself    bool
	TimeoutHappened  bool
	ForcefullyKilled bool
	BytesLogged      int64
}

type XReader struct{}

func (x *XReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("sss")
}

func (r *LongRun) Start() {
	if r.TimeToWaitForCleanup == 0 {
		r.TimeToWaitForCleanup = time.Second
	}
	if r.SplitFunc == nil {
		r.SplitFunc = ModScanLines //bufio.ScanLines
	}

	//	fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", r.Args)
	r.Cmd = exec.Command(r.Args[0], r.Args[1:]...)
	if r.WorkingDir != "" {
		r.Cmd.Dir = r.WorkingDir
	} else {
		r.Cmd.Dir = filepath.Dir(r.Args[0])
	}
	//	r.Cmd.Stdin = &XReader{}
	//	if isWindows := runtime.GOOS == "windows"; !isWindows {
	//		r.Cmd = exec.Command("sh", "-c", strings.Join(r.Args, " "))
	//	} else {
	//		cmds := append([]string{"/C", "start"}, r.Args...)
	//		r.Cmd = exec.Command("cmd", cmds...)
	//	}

	cmd := r.Cmd
	//cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	//var e1 error
	//var e2 error
	//	cmd.Stdout = os.Stdout
	//	cmd.Stderr = os.Stderr
	r.stdo, _ = cmd.StdoutPipe()
	r.stde, _ = cmd.StderrPipe()

	r.StartTime = time.Now().UTC()

	winHackProblemsChan := make(chan error, 1)
	// start!!!!!!!!
	cmdStopChan := runAsync_andGetErrorResultChan(func() error {
		// special hack for crashing of exe under windows
		// run is hanged, so the only hope is to get this done in another way
		go func() {
			defer func() { recover() }() // because of closing the chan that may be already closed by normal func
			for {
				if r.Duration != 0 {
					return
				}
				if cmd.ProcessState != nil {
					close(r.LogsProcessingIsOverChan)
					winHackProblemsChan <- errors.New("process exit state is not nil")
					break
				}
				time.Sleep(time.Second)
			}
		}()
		res := cmd.Run()

		return res
	})
	r.LogsProcessingIsOverChan = make(chan struct{})
	r.pipeLogsChannel(r.stdo, r.stde, r.LogsProcessingIsOverChan)

	timeoutChan := time.After(r.Timeout)
	if r.Timeout == 0 {
		timeoutChan = nil
	}

	//println("pumm....")

	//crashHangBug := false

	select {
	case code := <-cmdStopChan:
		r.ExitError = code
		r.StoppedItself = true
	case <-winHackProblemsChan:
		//		crashHangBug = true
		r.timeToStop()
	case <-interrupts.StopChannel:
		r.timeToStop()
	case <-timeoutChan:
		r.TimeoutHappened = true
		r.timeToStop()
	case <-r.StopChannel:
		r.timeToStop()
	}

	// note that timeToStop attempts to softly stop the app for X sec before killing the process
	// so real duration may be Timeout + WaitForCleanupTimeout
	r.Duration = time.Now().UTC().Sub(r.StartTime)

	r.WaitForFinish_ofLogProcessing()

	// hack for some reason, probably i'm missing something
	// but when process is killed then r.Cmd.ProcessState is empty
	ps := r.Cmd.ProcessState
	if r.Cmd.Process != nil && ps == nil {
		ps2, _ := r.Cmd.Process.Wait()
		if ps == nil {
			ps = ps2
		}
	}

	r.ProcessState = ps

	close(r.LogsChannel)
}

func (r *LongRun) GetOrCreateLogsChannel() chan *StreamData {
	if r.LogsChannel == nil {
		r.LogsChannel = make(chan *StreamData, 10000)
	}
	return r.LogsChannel
}

func (r *LongRun) timeToStop() {
	if r.Cmd != nil {
		p := r.Cmd.Process
		if p != nil {
			ForcefullyKilled := r.stopNoPanic()
			if !r.TimeoutHappened {
				r.ForcefullyKilled = ForcefullyKilled
			}
		}
	}
}

// returns true if app is interrupted
// may be used if Start was ran asynchronously
func (r *LongRun) WaitForFinish_ofLogProcessing() bool {
	select {
	case <-r.LogsProcessingIsOverChan:
		return false
	case <-interrupts.StopChannel:
		return true
	}
}

func (r *LongRun) Summary() string {
	e := ""
	if r.ExitError != nil {
		e = " with error [" + r.ExitError.Error() + "]"
	}
	pid := ""
	var systemTime time.Duration
	var userTime time.Duration

	ss := r.ProcessState
	if ss != nil {
		pid = fmt.Sprint(ss.Pid())
		systemTime = ss.SystemTime()
		userTime = ss.UserTime()
	}
	s := "Process [" + ArgsToString(r.Args) +
		"] (pid " + pid + ") was running " + fmt.Sprintf("%.2f", r.Duration.Seconds()) + "s" +
		" (systime:" + fmt.Sprintf("%.2f", systemTime.Seconds()) + "s, usertime:" + fmt.Sprintf("%.2f", userTime.Seconds()) + "s) and" +
		" exited" +
		e +
		If(r.ForcefullyKilled).Then(", was force-killed").Else("").Str() +
		If(r.StoppedItself).Then(", stopped itself").Else("").Str() +
		If(r.TimeoutHappened).Then(", KILLED for running too long ("+fmt.Sprintf("%.2f", r.Timeout.Seconds())+"s)").Else("").Str() +
		", logs amount is " + strconv.Itoa(int(r.BytesLogged)) + " bytes"

	return s
}

//--------------------------------

// proper stopping of the child
// 1) attempt to send http request
// 2) attempt to kill via interrupt
// 3) attempt to force kill using cmd.Process.Kill()
// function blocks while process is alive
// returns true if process was forcefully killed
func (r *LongRun) stopNoPanic() bool {
	cmd := r.Cmd
	HttpRestartUrl := r.ShutdownURL
	defer func() { recover() }()
	go cmd.Process.Signal(os.Interrupt)
	if HttpRestartUrl != "" && HttpRestartUrl != "-" {
		go GetHttpContents(HttpRestartUrl)
	}
	ch := make(chan int, 3)
	go func() {
		defer func() { recover() }()
		time.Sleep(r.TimeToWaitForCleanup)
		pid := cmd.Process.Pid
		if runtime.GOOS == "windows" {

			if killcmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)); killcmd != nil {
				if e := killcmd.Run(); e != nil {
					//println(e)
				}
				time.Sleep(time.Second)
			}

		} else {
			if killcmd := exec.Command("pkill", "-TERM", "-P", strconv.Itoa(pid)); killcmd != nil {
				if e := killcmd.Run(); e != nil {
					//println(e)
				}
				time.Sleep(time.Second)
			}
		}

		cmd.Process.Kill()

		ch <- 2
	}()
	go func() {
		defer func() { recover() }()
		cmd.Process.Wait()
		ch <- 1
	}()

	// wait for some event to happen quickly
	how := <-ch
	return how == 2
}

// read data from reader and put it into the channel
func (r *LongRun) readToChannel(a chan *StreamData, cn chan int64, reader io.ReadCloser, isStderr bool) {
	var byteCount int64 = 0
	defer func() { recover() }()
	defer func() {
		//a <- &StreamData{isStderr, nil}
		defer func() { recover() }()
		if cn != nil {
			cn <- byteCount
		}
	}()
	if reader == nil {
		return
	}
	scanner := bufio.NewScanner(reader)
	scanner.Split(r.SplitFunc) // bufio.ScanLines
	for scanner.Scan() {
		b := scanner.Bytes()
		byteCount += int64(len(b))
		//println("... " + string(b))
		a <- &StreamData{isStderr, string(b)}
	}
}

// pipe channel that is filled with stdout and stderr data in background
func (r *LongRun) pipeLogsChannel(stdout, stderr io.ReadCloser, logsProcessingIsOverChan chan struct{}) {
	ch := r.LogsChannel //make(chan *StreamData, 100000)
	auxCloseNotifyChan := make(chan int64, 2)
	go func() {
		// we do recover because of closing the chan that may be already closed...
		// usually it is not, but because of bug of strange Wait hang under windows
		// when app is crashed
		defer func() { recover() }()
		var b int64 = 0
		b += <-auxCloseNotifyChan
		b += <-auxCloseNotifyChan
		r.BytesLogged = b
		if logsProcessingIsOverChan != nil {
			close(logsProcessingIsOverChan)
		}
		//close(ch) -- will be done on Start func end

	}()
	go r.readToChannel(ch, auxCloseNotifyChan, stdout, false)
	go r.readToChannel(ch, auxCloseNotifyChan, stderr, true)
}

func runAsync_andGetErrorResultChan(f func() error) chan error {
	itStopped := make(chan error, 2)
	go func() {
		res := f()
		itStopped <- res
	}()

	return itStopped
}
