package pumpx2

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// serveTimeout bounds one request to the serve process. A parse or encode takes
// milliseconds once the JVM is warm; anything near this is a wedged process.
const serveTimeout = 30 * time.Second

// serveEnvKeys are the variables a one-shot cliparser command reads from the
// process environment. The serve process inherits nothing, so each request
// carries whichever of them faketandem's own environment sets, which is what a
// one-shot JarRunner subprocess would have inherited.
var serveEnvKeys = []string{
	"PUMP_AUTHENTICATION_KEY",
	"PUMP_PAIRING_CODE",
	"PUMP_TIME_SINCE_RESET",
	"PUMPX2_MAX_CHUNK_SIZE",
}

var errServeUnsupported = errors.New("cliparser jar has no serve command")

// ServeRunner keeps one `cliparser serve` JVM running and sends it every parse
// and encode, instead of starting a JVM per message (~0.4 s each). A jar that
// predates serve is detected on the first request, and every call then goes
// through fallback.
type ServeRunner struct {
	jarPath  string
	javaCmd  string
	fallback *JarRunner

	mu          sync.Mutex
	unsupported bool
	proc        *serveProcess
	nextID      int
}

type serveProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	// served is set once the process has answered a request, which is what
	// tells a jar without serve (it exits at once) from a crash.
	served bool
}

type serveRequest struct {
	ID   int               `json:"id"`
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

type serveReply struct {
	ID     int    `json:"id"`
	OK     bool   `json:"ok"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error"`
}

// NewServeRunner creates a runner for the given cliparser jar. The JVM starts on
// the first request.
func NewServeRunner(jarPath, javaCmd string) *ServeRunner {
	return &ServeRunner{
		jarPath:  jarPath,
		javaCmd:  javaCmd,
		fallback: NewJarRunner(jarPath, javaCmd),
	}
}

// Parse decodes a message from its raw BLE fragments -- see JarRunner.Parse.
func (r *ServeRunner) Parse(btChar string, rawPacketsHex []string) (string, error) {
	env := processEnv()
	if btChar != "" {
		env["PUMPX2_CHARACTERISTIC"] = btChar
	}
	output, err := r.call([]string{"parse", strings.Join(rawPacketsHex, " ")}, env)
	if errors.Is(err, errServeUnsupported) {
		return r.fallback.Parse(btChar, rawPacketsHex)
	}
	return output, err
}

// Encode builds a message -- see JarRunner.Encode.
func (r *ServeRunner) Encode(txID int, messageName string, params map[string]interface{}) (string, error) {
	paramsJSON, err := encodeParamsJSON(params)
	if err != nil {
		return "", err
	}
	output, err := r.call([]string{"encode", fmt.Sprintf("%d", txID), messageName, paramsJSON}, processEnv())
	if errors.Is(err, errServeUnsupported) {
		return r.fallback.Encode(txID, messageName, params)
	}
	return output, err
}

// Close stops the serve process, if one is running.
func (r *ServeRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked()
	return nil
}

func processEnv() map[string]string {
	env := map[string]string{}
	for _, key := range serveEnvKeys {
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}
	return env
}

func (r *ServeRunner) call(args []string, env map[string]string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.unsupported {
		return "", errServeUnsupported
	}

	// One retry: parse and encode have no side effects, and a JVM that died since
	// the last request is restarted by the first attempt's cleanup.
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		reply, err := r.roundTripLocked(args, env)
		if err == nil {
			if !reply.OK {
				return "", fmt.Errorf("cliparser %s failed: %s\nStderr: %s", args[0], reply.Error, reply.Stderr)
			}
			return reply.Stdout, nil
		}
		if errors.Is(err, errServeUnsupported) {
			log.Warnf("cliparser jar %s has no serve command; starting a JVM per message instead", r.jarPath)
			r.unsupported = true
			return "", err
		}
		log.Warnf("cliparser serve process failed (%v); restarting it", err)
		lastErr = err
	}
	return "", lastErr
}

func (r *ServeRunner) roundTripLocked(args []string, env map[string]string) (*serveReply, error) {
	if r.proc == nil {
		if err := r.startLocked(); err != nil {
			return nil, err
		}
	}
	proc := r.proc

	r.nextID++
	request := serveRequest{ID: r.nextID, Args: args, Env: env}
	line, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal serve request: %w", err)
	}

	type result struct {
		reply *serveReply
		err   error
	}
	done := make(chan result, 1)
	go func() {
		if _, err := proc.stdin.Write(append(line, '\n')); err != nil {
			done <- result{err: err}
			return
		}
		replyLine, err := proc.stdout.ReadBytes('\n')
		if err != nil {
			done <- result{err: err}
			return
		}
		var reply serveReply
		if err := json.Unmarshal(replyLine, &reply); err != nil {
			done <- result{err: fmt.Errorf("unreadable serve reply %q: %w", replyLine, err)}
			return
		}
		done <- result{reply: &reply}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			served := proc.served
			r.stopLocked()
			if !served {
				return nil, errServeUnsupported
			}
			return nil, res.err
		}
		if res.reply.ID != request.ID {
			r.stopLocked()
			return nil, fmt.Errorf("serve reply id %d for request %d", res.reply.ID, request.ID)
		}
		proc.served = true
		return res.reply, nil
	case <-time.After(serveTimeout):
		r.stopLocked()
		return nil, fmt.Errorf("no serve reply within %s", serveTimeout)
	}
}

func (r *ServeRunner) startLocked() error {
	cmd := exec.Command(r.javaCmd, "-jar", r.jarPath, "serve")
	cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("serve stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("serve stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("serve stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cliparser serve: %w", err)
	}
	// Drained so the JVM never blocks on a full stderr pipe; pumpX2's own logging goes there.
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Tracef("cliparser serve: %s", scanner.Text())
		}
	}()
	log.Debugf("Started cliparser serve (pid %d)", cmd.Process.Pid)
	r.proc = &serveProcess{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout)}
	return nil
}

func (r *ServeRunner) stopLocked() {
	if r.proc == nil {
		return
	}
	proc := r.proc
	r.proc = nil
	_ = proc.stdin.Close() // Safe to ignore: the process is being discarded.
	if proc.cmd.Process != nil {
		_ = proc.cmd.Process.Kill() // Safe to ignore: it may already have exited.
	}
	go func() {
		_ = proc.cmd.Wait() // Safe to ignore: reaps the process; its exit status is not used.
	}()
}
