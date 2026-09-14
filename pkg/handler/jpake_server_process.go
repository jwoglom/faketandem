package handler

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
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

	// mu guards stderrTail -- the most recent lines jpake-server wrote to
	// stderr, since when the JVM dies mid-handshake its Java stack trace is
	// the only evidence of why and it goes to stderr -- and readErr, which
	// records why reading stdout stopped.
	//
	// Both exist so a failed handshake carries its own explanation instead of
	// leaving it buried at debug level in a log nobody captured.
	mu         sync.Mutex
	stderrTail []string
	readErr    error

	closeOnce sync.Once
}

// jpakeServerStderrTailLines bounds how much of jpake-server's stderr is kept
// for the failure report. A Java stack trace is a few dozen lines; cliparser
// is also chatty at parse time, so keep enough to cover a trace plus context.
const jpakeServerStderrTailLines = 200

// jpakeServerLineBuffer bounds a single line of jpake-server output. A JPAKE
// envelope with its packets array is a couple of kilobytes; 1 MiB is far
// beyond anything legitimate.
const jpakeServerLineBuffer = 1 << 20

// ErrJPAKEServerFailed marks a handshake that cannot be completed because
// pumpX2's jpake-server gave up: it printed its own {"error": ...} envelope,
// or it died, mid-handshake. It is a distinct sentinel because the recovery is
// distinct -- nothing this process does can revive that handshake, so the
// right move is to fail the round immediately, throw the dead server away, and
// let the client re-pair against a fresh one, rather than sit on a stalled
// subprocess until the driver's own 30-90 second timeout.
var ErrJPAKEServerFailed = errors.New("pumpX2 jpake-server failed mid-handshake")

// jpakeServerErrorLineRegex matches the JSON envelope jpake-server prints when
// it gives up: {"error":"..."}. Main.jpakeAuthServer returns that string and
// main() prints it as the process's last line, so it is both a definitive
// failure signal and the only description of what went wrong on stdout.
var jpakeServerErrorLineRegex = regexp.MustCompile(`\{"error"\s*:\s*"((?:[^"\\]|\\.)*)"`)

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
	go p.stderrLoop(stderr)

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
		// Notably bufio.ErrTooLong, if a line ever exceeded
		// jpakeServerLineBuffer: that ends the stream silently and would
		// otherwise be indistinguishable from the process exiting.
		log.Debugf("jpake-server stdout read ended: %v", err)

		p.mu.Lock()
		p.readErr = err
		p.mu.Unlock()
	}
}

// stderrLoop drains stderr so a chatty JVM can never fill its pipe buffer and
// wedge the handshake, logs it for debugging, and keeps the tail of it for
// recentStderr.
func (p *jpakeServerProcess) stderrLoop(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), jpakeServerLineBuffer)
	for scanner.Scan() {
		line := scanner.Text()
		log.Debugf("jpake-server stderr: %s", line)

		p.mu.Lock()
		p.stderrTail = append(p.stderrTail, line)
		if len(p.stderrTail) > jpakeServerStderrTailLines {
			p.stderrTail = p.stderrTail[len(p.stderrTail)-jpakeServerStderrTailLines:]
		}
		p.mu.Unlock()
	}
}

// recentStderr returns the tail of what jpake-server wrote to stderr, as a
// single newline-joined string (empty if it said nothing).
func (p *jpakeServerProcess) recentStderr() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return strings.Join(p.stderrTail, "\n")
}

// readFailure returns why reading jpake-server's stdout stopped, if it stopped
// for any reason other than the process closing it.
func (p *jpakeServerProcess) readFailure() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.readErr
}

// send writes one line to jpake-server's stdin.
func (p *jpakeServerProcess) send(line string) error {
	if _, err := io.WriteString(p.stdin, line+"\n"); err != nil {
		return fmt.Errorf("%w: writing to its stdin failed: %v%s", ErrJPAKEServerFailed, err, p.stderrSuffix())
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
				if readErr := p.readFailure(); readErr != nil {
					return nil, fmt.Errorf("%w: reading its stdout failed while waiting for a line matching %s: %v%s",
						ErrJPAKEServerFailed, re, readErr, p.stderrSuffix())
				}
				return nil, fmt.Errorf("%w: it exited before printing a line matching %s%s",
					ErrJPAKEServerFailed, re, p.stderrSuffix())
			}
			if matches := re.FindStringSubmatch(line); matches != nil {
				return matches, nil
			}
			// An {"error": ...} envelope is jpake-server's last word: it has
			// already abandoned the handshake and is about to exit. Failing on
			// it here rather than waiting for EOF or the timeout is what turns
			// a stalled pairing into an immediate, explained failure.
			if errMatch := jpakeServerErrorLineRegex.FindStringSubmatch(line); errMatch != nil {
				return nil, fmt.Errorf("%w: %s%s%s",
					ErrJPAKEServerFailed, errMatch[1], diagnoseJPAKEServerError(errMatch[1], re), p.stderrSuffix())
			}
			log.Debugf("jpake-server (skipped): %s", line)
		case <-deadline.C:
			return nil, fmt.Errorf("timed out after %s waiting for jpake-server output matching %s%s",
				timeout, re, p.stderrSuffix())
		}
	}
}

// stderrSuffix renders the tail of jpake-server's stderr for inclusion in an
// error message, or "" if it said nothing. jpake-server prints the Java stack
// trace of an aborted handshake there and nowhere else, so without this a
// failure reaches the log as a bare timeout with no cause attached.
func (p *jpakeServerProcess) stderrSuffix() string {
	tail := p.recentStderr()
	if tail == "" {
		return ""
	}
	return "; jpake-server stderr tail:\n" + tail
}

// diagnoseJPAKEServerError names the known upstream cause behind one of
// jpake-server's error envelopes, so the log says what happened instead of
// leaving an operator to work backwards from "null".
//
// "Exception during server JPAKE authentication: null" while we are waiting
// for JPAKE_2 is pumpX2's own 1-in-256 dice roll, not anything the emulator
// did. jpake-server builds its round-2 message with
// io.particle.crypto.EcJpake.getRound2(), whose writeNum() encodes the
// zero-knowledge-proof scalar r with a *minimal-length* unsigned encoding
// (BigIntegers.asUnsignedByteArray). r is uniform mod n, so about one time in
// 256 its top byte is zero and the encoding is 31 bytes instead of 32, making
// the whole round-2 blob 167 bytes instead of 168. Jpake2Response.parse then
// rejects it (Validate.isTrue(raw.length == props().size()), 170 = 2 + 168),
// the exception surfaces through reflection as an InvocationTargetException
// -- whose getMessage() is null, hence the "null" -- and jpake-server prints
// the envelope and exits without ever sending JPAKE_2.
//
// Nothing on this side can rescue that handshake: the client already holds the
// dead server's round-1 values, so a replacement server's round 1 would not
// match. The client has to pair again, which spawns a fresh jpake-server whose
// next roll of r is almost certainly fine.
func diagnoseJPAKEServerError(errText string, waitingFor *regexp.Regexp) string {
	if !strings.Contains(errText, "Exception during server JPAKE authentication: null") {
		return ""
	}
	if !strings.Contains(waitingFor.String(), "JPAKE_2") {
		return ""
	}
	return " (known pumpX2 bug: EcJpake.getRound2 encodes the round-2 ZKP scalar" +
		" with a minimal-length unsigned encoding, so roughly 1 handshake in 256" +
		" produces a 167-byte round 2 that Jpake2Response -- which requires exactly" +
		" 168 -- rejects. This handshake cannot be recovered; the client must pair" +
		" again, which starts a fresh jpake-server)"
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
