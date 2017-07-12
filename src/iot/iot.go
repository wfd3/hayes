package main

import (
	"io"
	"os"
)

// Implements io.ReadWriteCloser
type myReadWriter struct {
	in *os.File
	out *os.File
}

func (m myReadWriter) Read(p []byte) (int, error) {
	return m.in.Read(p)
}

func (m myReadWriter) Write(p []byte) (int, error) {
	return m.out.Write(p)
}

func (m myReadWriter) Close() error {
	err1 := m.in.Close()
	err2 := m.out.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func do() (io.ReadWriteCloser) {
	var q myReadWriter
	q.in = os.Stdin
	q.out = os.Stdout

	return io.ReadWriteCloser(q)
}

func main() {
	var b []byte
	b = make([]byte, 100)
	
	f := do()
	f.Write([]byte("magic!\n"))
	f.Read(b)
	f.Write(b)
	if err := f.Close(); err != nil {
		panic(err)
	}

}
