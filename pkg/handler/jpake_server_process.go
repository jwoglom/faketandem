package handler

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// jpakeServerProcess drives pumpX2's cliparser "jpake-server" subprocess over
// plain stdin/stdout pipes.
//
// It deliberately does NOT use a pseudo-terminal. The previous implementation
// spawned jpake-server through goexpect, whose SpawnWithArgs allocates a pty
// via goterm's OpenPTY -> ioctl(TIOCSPTLCK), which exists only on Linux: on
// macOS every spawn failed with "inappropriate ioctl for device", so a real
// Jpake1aRequest went unanswered and the driver timed out after 30 seconds.
//
// jpake-server needs nothing a terminal provides. Its protocol is strictly
// line-oriented -- it reads one whitespace-separated line of raw BLE fragments
// per client message from stdin and prints one "JPAKE_xx: {json}" line per
// server message to stdout -- which is exactly how the reference end-to-end
// test (TestPumpX2JPAKEAuthenticator_FullFlow) has always talked to it. Pipes
// behave identically on Linux and macOS.
type jpakeServerProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	// lines carries stdout lines in arrival order. It is buffered so the
	// reader goroutine never blocks on a caller that is busy elsewhere; a line
	// that arrives before anyone asks for it waits here rather than being
	// lost, which is what the pty-buffer behavior used to provide.
	lines chan string

	closeOnce sync.Once
}

// jpakeServerLineBuffer bounds a single line of jpake-server output. A JPAKE
// envelope with its packets array is a couple of kilobytes; 1 MiB is far
// beyond anything legitimate.
const jpakeServerLineBuffer = 1 << 20

// startJPAKEServer launches name with args and wires up its pipes.
func startJPAKEServer(name string, args []string) (*jpakeServerProcess, error) {
	cmd := exec.Command(name, args...) //nolint:gosec // the command is operator-configured (-java-cmd/-pumpx2-path), not user input

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create jpake-server stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create jpake-server stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create jpake-server stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start jpake-server process: %w", err)
	}

	p := &jpakeServerProcess{
		cmd:   cmd,
		stdin: stdin,
		lines: make(chan string, 256),
	}

	go p.readLoop(stdout)
	go logJPAKEServerStderr(stderr)

	return p, nil
}

// readLoop feeds stdout lines into p.lines and closes it at EOF, so a waiting
// expect() learns that the process died instead of hanging until its timeout.
func (p *jpakeServerProcess) readLoop(stdout io.Reader) {
	defer close(p.lines)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), jpakeServerLineBuffer)
	for scanner.Scan() {
		p.lines <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Debugf("jpake-server stdout read ended: %v", err)
	}
}

// logJPAKEServerStderr drains stderr so a chatty JVM can never fill its pipe
// buffer and wedge the handshake, and surfaces what it said for debugging.
func logJPAKEServerStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), jpakeServerLineBuffer)
	for scanner.Scan() {
		log.Debugf("jpake-server stderr: %s", scanner.Text())
	}
}

// send writes one line to jpake-server's stdin.
func (p *jpakeServerProcess) send(line string) error {
	if _, err := io.WriteString(p.stdin, line+"\n"); err != nil {
		return fmt.Errorf("failed to write to jpake-server stdin: %w", err)
	}
	return nil
}

// expect reads stdout lines until one matches re, and returns its submatches.
// Lines that do not match are logged and discarded, which is what makes the
// JVM's own startup chatter harmless.
func (p *jpakeServerProcess) expect(re *regexp.Regexp, timeout time.Duration) ([]string, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				return nil, fmt.Errorf("jpake-server exited before printing a line matching %s", re)
			}
			if matches := re.FindStringSubmatch(line); matches != nil {
				return matches, nil
			}
			log.Debugf("jpake-server (skipped): %s", line)
		case <-deadline.C:
			return nil, fmt.Errorf("timed out after %s waiting for jpake-server output matching %s", timeout, re)
		}
	}
}

// close terminates the subprocess and reaps it.
//
// Killing rather than waiting for a clean exit is deliberate: jpake-server
// blocks reading stdin between rounds, and an abandoned handshake would
// otherwise leave a JVM running for the lifetime of the emulator (the leak
// JPAKESessionManager.Remove/RemoveAll now call this to prevent). Closing
// stdin first gives a server that is mid-handshake the chance to exit on its
// own; the kill covers the rest.
//
// Every failure along the way is logged rather than returned: by the time this
// runs the handshake is over one way or another, and there is nothing a caller
// could do differently about a JVM that had already exited.
func (p *jpakeServerProcess) close() {
	p.closeOnce.Do(func() {
		if cerr := p.stdin.Close(); cerr != nil {
			log.Debugf("Error closing jpake-server stdin: %v", cerr)
		}
		if p.cmd.Process != nil {
			if kerr := p.cmd.Process.Kill(); kerr != nil {
				log.Debugf("Error killing jpake-server process: %v", kerr)
			}
		}
		// Wait reaps the process and releases the pipes. The error is the
		// "signal: killed" we just caused, so it is not worth reporting.
		if werr := p.cmd.Wait(); werr != nil {
			log.Debugf("jpake-server exited: %v", werr)
		}
	})
}
