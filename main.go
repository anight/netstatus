package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"
)

func pinger(c chan string) {

	last_response := time.Now()

	for {
		var stdout, stderr io.ReadCloser
		var cmd *exec.Cmd

		for {
			cmd = exec.Command("ping", "8.8.8.8")
			var err error
			stdout, err = cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}
			stderr, err = cmd.StderrPipe()
			if err != nil {
				log.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				c <- fmt.Sprintf("ping has failed to start: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}

		//		c <- fmt.Sprintf("ping is running")

		line_reader := func(r io.ReadCloser) chan string {
			c := make(chan string, 10)

			go func(r io.ReadCloser, c chan string) {
				reader := bufio.NewReader(r)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						//						c <- fmt.Sprintf("read error: %v", err)
						break
					}
					line = strings.TrimRight(line, "\n")
					c <- line
				}
				close(c)
			}(r, c)

			return c
		}

		stdout_msg_reader := line_reader(stdout)
		stderr_msg_reader := line_reader(stderr)

		last_message := time.Now()
		last_message_body := ""

	process_done:
		for {
			// todo: add "nmcli connection show --active" output

			select {
			case msg, ok := <-stdout_msg_reader:
				if !ok {
					break process_done
				}
				last_message_body = msg
				response := last_message_body
				if strings.Contains(msg, "Unreachable") {
					no_response := time.Since(last_response).Seconds()
					if no_response > 5 {
						response += fmt.Sprintf(", no network for last %v seconds", int(no_response))
					}
				} else {
					last_response = time.Now()
				}
				c <- response
				last_message = time.Now()
			case msg, ok := <-stderr_msg_reader:
				if !ok {
					break process_done
				}
				last_message_body = fmt.Sprintf("stderr: %s", msg)
				response := last_message_body
				no_response := time.Since(last_response).Seconds()
				if no_response > 5 {
					response += fmt.Sprintf(", no network for last %v seconds", int(no_response))
				}
				c <- response
				last_message = time.Now()
			case <-time.After(1200 * time.Millisecond):
				no_messages := time.Since(last_message).Seconds()
				if no_messages > 1 {
					no_response := time.Since(last_response).Seconds()
					if no_response > 5 {
						response := last_message_body
						if response != "" {
							response += ", "
						}
						response += fmt.Sprintf("no network for last %v seconds", int(no_response))
						c <- response
					}
				}
			}
		}

		cmd.Wait()
		//		c <- fmt.Sprintf("ping has finished, restarting")
		time.Sleep(1 * time.Second)
	}

}

func main() {
	status := make(chan string, 10)
	go pinger(status)
	for {
		s := <-status
		fmt.Printf("%s\033[K\r", s)
	}
}
