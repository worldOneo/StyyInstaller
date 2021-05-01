package main

import (
	"io"
)

type WriteCounter struct {
	total      int64
	expect     int64
	writer     io.Writer
	updateFunc func(float64)
}

func NewWriteCounter(e int64, wrtr io.Writer, update func(float64)) *WriteCounter {
	return &WriteCounter{0, e, wrtr, update}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	add, err := wc.writer.Write(p)
	if err != nil {
		return add, err
	}
	wc.total += int64(add)
	wc.updateFunc(float64(wc.total) / float64(wc.expect))
	return add, nil
}

func (wc *WriteCounter) WriteFullFrom(rdr io.Reader) error {
	if _, err := io.Copy(wc, rdr); err != nil {
		return err
	}
	return nil
}
