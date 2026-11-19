package app

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// One Lootnest per user on Linux: the first one listens on an abstract
// Unix socket (no file to clean up; it goes when the process does), and a
// second one hands it its request ("show", "launch:<id>", "safe") and quits.

var instanceLn net.Listener

func instanceAddr() string { return "@lootnest-" + strconv.Itoa(os.Getuid()) }

// AcquireInstance makes this the running Lootnest; false when another one is.
func AcquireInstance() bool {
	ln, err := net.Listen("unix", instanceAddr())
	if err != nil {
		return false
	}
	instanceLn = ln
	return true
}

// SendToRunning hands a command to the running Lootnest.
func SendToRunning(cmd string) bool {
	c, err := net.DialTimeout("unix", instanceAddr(), 2*time.Second)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = c.Write([]byte(cmd + "\n"))
	return err == nil
}

// serveInstance takes commands from later Lootnests, one line each, for as
// long as this one runs.
func serveInstance(a *App) {
	if instanceLn == nil {
		return
	}
	go func() {
		for {
			c, err := instanceLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				line, err := bufio.NewReader(c).ReadString('\n')
				cmd := strings.TrimSpace(line)
				if err != nil || cmd == "" || len(cmd) > 512 {
					return
				}
				// Only what a second Lootnest asks for; Steam's requests
				// and quit-now stay Windows' own channel.
				if cmd == "show" || cmd == "safe" || strings.HasPrefix(cmd, "launch:") {
					a.ui(func() { a.handleCommand(cmd) })
				}
			}(c)
		}
	}()
}
