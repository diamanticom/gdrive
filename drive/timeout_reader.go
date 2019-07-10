package drive

import (
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"
)

const TimeoutTimerInterval = time.Second * 10

type timeoutReaderWrapper func(io.Reader) io.Reader

func getTimeoutReaderWrapperContext(timeout time.Duration) (timeoutReaderWrapper, context.Context) {
	ctx, cancel := context.WithCancel(context.TODO())
	wrapper := func(r io.Reader) io.Reader {
		// Return untouched reader if timeout is 0
		if timeout == 0 {
			return r
		}

		return getTimeoutReader(r, cancel, timeout)
	}
	return wrapper, ctx
}

func getTimeoutReaderContext(r io.Reader, timeout time.Duration) (io.Reader, context.Context) {
	ctx, cancel := context.WithCancel(context.TODO())

	// Return untouched reader if timeout is 0
	if timeout == 0 {
		return r, ctx
	}

	return getTimeoutReader(r, cancel, timeout), ctx
}

func getTimeoutReader(r io.Reader, cancel context.CancelFunc, timeout time.Duration) io.Reader {
	return &TimeoutReader{
		reader:         r,
		cancel:         cancel,
		mutex:          &sync.Mutex{},
		maxIdleTimeout: timeout,
	}
}

type TimeoutReader struct {
	reader         io.Reader
	cancel         context.CancelFunc
	lastActivity   time.Time
	timer          *time.Timer
	mutex          *sync.Mutex
	maxIdleTimeout time.Duration
	done           bool
}

func (t *TimeoutReader) Read(p []byte) (int, error) {
	if t.timer == nil {
		t.startTimer()
	}

	t.mutex.Lock()

	// Read
	n, err := t.reader.Read(p)

	t.lastActivity = time.Now()
	t.done = err != nil

	t.mutex.Unlock()

	if t.done {
		t.stopTimer()
	}

	return n, err
}

func (t *TimeoutReader) startTimer() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.done {
		t.timer = time.AfterFunc(TimeoutTimerInterval, t.timeout)
	}
}

func (t *TimeoutReader) stopTimer() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *TimeoutReader) timeout() {
	t.mutex.Lock()

	if t.done {
		t.mutex.Unlock()
		return
	}

	if time.Since(t.lastActivity) > t.maxIdleTimeout {
		t.cancel()
		t.mutex.Unlock()
		return
	}

	t.mutex.Unlock()
	t.startTimer()
}
