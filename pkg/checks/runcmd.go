package checks

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

func runCmd(cmd string, args []string) (string, string, error) {
	c := exec.Command(cmd, args...)
	//c.Stdin = os.Stdin
	//var out bytes.Buffer
	//cmd.Stdout = &out

	stdoutCh := make(chan string, 10)
	fulloutCh := make(chan string, 10)

	wg := sync.WaitGroup{}

	wgFullout := sync.WaitGroup{}
	wgFullout.Add(2)

	stdoutW := func() *io.PipeWriter {
		stdoutR, stdoutW := io.Pipe()

		stdoutScanner := bufio.NewScanner(stdoutR)

		c.Stdout = stdoutW

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(stdoutCh)
			defer wgFullout.Done()

			for stdoutScanner.Scan() {
				text := stdoutScanner.Text() + "\n"
				stdoutCh <- text
				fulloutCh <- text
				fmt.Fprintf(os.Stdout, text)
			}
		}()

		return stdoutW
	}()

	stderrW := func() *io.PipeWriter {
		stderrR, stderrW := io.Pipe()

		stderrScanner := bufio.NewScanner(stderrR)

		c.Stderr = stderrW

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer wgFullout.Done()

			for stderrScanner.Scan() {
				text := stderrScanner.Text() + "\n"
				fulloutCh <- text
				fmt.Fprintf(os.Stderr, text)
			}
		}()

		return stderrW
	}()

	go func() {
		wgFullout.Wait()

		close(fulloutCh)
	}()

	stdout := func() *bytes.Buffer {
		buf := &bytes.Buffer{}

		wg.Add(1)
		go func() {
		FOR:
			for {
				select {
				case v, ok := <-stdoutCh:
					if ok {
						buf.WriteString(v)
					} else {
						break FOR
					}
				}
			}
			wg.Done()
		}()

		return buf
	}()

	fullout := func() *bytes.Buffer {
		buf := &bytes.Buffer{}

		wg.Add(1)
		go func() {
		FOR:
			for {
				select {
				case v, ok := <-fulloutCh:
					if ok {
						buf.WriteString(v)
					} else {
						break FOR
					}
				}
			}
			wg.Done()
		}()

		return buf
	}()

	err := c.Run()

	// As the command returned, the pipes should be safe to close now.
	// Note that you have to close write-side of pipes. If you close readers first, you'll end up seeing "write to closed pipes" errors
	stdoutW.Close()
	stderrW.Close()

	// Ensure that:
	// - stdout and stderr are sent to the channels and
	// - all the messages sent to the channels are consumed and
	// - their contents are stored in buffers
	wg.Wait()

	return stdout.String(), fullout.String(), err
}

