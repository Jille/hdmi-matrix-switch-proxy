// Hacky proxy to make my browser talk to my HDMI Matrix Switch properly.
package main

import (
	"io"
	"log"
	"net"
	"regexp"
	"strings"
)

func main() {
	sock, err := net.Listen("tcp4", ":8080")
	if err != nil {
		panic(err)
	}

	for {
		s, err := sock.Accept()
		if err != nil {
			log.Printf("Accept failed: %v", err)
			continue
		}
		go handle(s)
	}
}

func handle(s net.Conn) {
	defer s.Close()
	if err := doHandle(s); err != nil {
		log.Printf("Connection handling failed: %v", err)
	} else {
		log.Printf("Connection died normally")
	}
}

func doHandle(s net.Conn) error {
	c, err := net.Dial("tcp4", "192.168.0.65:80")
	if err != nil {
		return err
	}
	defer c.Close()
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		n, err := c.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		rw := &replaceWriter{s}
		if !strings.HasPrefix(string(buf), "HTTP/") {
			_, err := io.WriteString(rw, "HTTP/1.0 200 Whatever\r\n\r\n")
			if err != nil {
				errCh <- err
				return
			}
		}
		rw.Write(buf[:n])
		buf = nil
		_, err = io.Copy(rw, c)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(c, s)
		errCh <- err
	}()
	return <-errCh
}

type replaceWriter struct {
	target io.Writer
}

var re = regexp.MustCompile(`Content-Length: \d+\r\n`)

func (rw *replaceWriter) Write(b []byte) (int, error) {
	nb := re.ReplaceAll(b, nil)
	stripped := len(b) - len(nb)
	n, err := rw.target.Write(nb)
	return n + stripped, err
}
