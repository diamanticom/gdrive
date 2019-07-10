package drive

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"
)

const MaxDrawInterval = time.Second * 1
const MaxRateInterval = time.Second * 3

func getProgressReader(r io.Reader, w io.Writer, size int64) io.Reader {
	// Don't wrap reader if output is discarded or size is too small
	if w == ioutil.Discard || (size > 0 && size < 1024*1024) {
		return r
	}

	return &Progress{
		Reader: r,
		Writer: w,
		Size:   size,
	}
}

type Progress struct {
	Writer       io.Writer
	Reader       io.Reader
	Size         int64
	progress     int64
	rate         int64
	rateProgress int64
	rateUpdated  time.Time
	updated      time.Time
	done         bool
}

func (pr *Progress) Read(p []byte) (int, error) {
	// Read
	n, err := pr.Reader.Read(p)

	now := time.Now()
	isLast := err != nil

	// Increment progress
	newProgress := pr.progress + int64(n)
	pr.progress = newProgress

	// Initialize rate state
	if pr.rateUpdated.IsZero() {
		pr.rateUpdated = now
		pr.rateProgress = newProgress
	}

	// Update rate every x seconds
	if pr.rateUpdated.Add(MaxRateInterval).Before(now) {
		pr.rate = calcRate(newProgress-pr.rateProgress, pr.rateUpdated, now)
		pr.rateUpdated = now
		pr.rateProgress = newProgress
	}

	// Draw progress every x seconds
	if pr.updated.Add(MaxDrawInterval).Before(now) || isLast {
		pr.draw(isLast)
		pr.updated = now
	}

	// Mark as done if error occurs
	pr.done = isLast

	return n, err
}

func (pr *Progress) draw(isLast bool) {
	if pr.done {
		return
	}

	pr.clear()

	// Print progress
	_, _ = fmt.Fprintf(pr.Writer, "%s", formatSize(pr.progress, false))

	// Print total size
	if pr.Size > 0 {
		_, _ = fmt.Fprintf(pr.Writer, "/%s", formatSize(pr.Size, false))
	}

	// Print rate
	if pr.rate > 0 {
		_, _ = fmt.Fprintf(pr.Writer, ", Rate: %s/s", formatSize(pr.rate, false))
	}

	if isLast {
		pr.clear()
	}
}

func (pr *Progress) clear() {
	_, _ = fmt.Fprintf(pr.Writer, "\r%50s\r", "")
}
