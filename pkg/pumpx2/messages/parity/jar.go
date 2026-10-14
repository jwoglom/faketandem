package parity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// JVMUTF8Flags make the JVM print UTF-8 whatever the locale: under a POSIX
// locale it prints every non-ASCII character of a toString as '?'. The first
// covers JDK 17, the others JDK 18 and later.
var JVMUTF8Flags = []string{"-Dfile.encoding=UTF-8", "-Dstdout.encoding=UTF-8", "-Dsun.stdout.encoding=UTF-8"}

// Jar drives one `cliparser serve` process: the same jar entry point
// faketandem's ServeRunner uses, without depending on that package.
type Jar struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID int
}

// StartJar starts `java -jar <jarPath> serve`.
func StartJar(javaCmd, jarPath string) (*Jar, error) {
	cmd := exec.Command(javaCmd, append(JVMUTF8Flags, "-jar", jarPath, "serve")...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Jar{cmd: cmd, stdin: stdin, stdout: bufio.NewReaderSize(stdout, 1<<20)}, nil
}

// Close stops the JVM.
func (j *Jar) Close() {
	_ = j.stdin.Close()      // Safe to ignore: the process is being discarded.
	_ = j.cmd.Process.Kill() // Safe to ignore: it may already have exited.
	_ = j.cmd.Wait()         // Safe to ignore: reaps the process.
}

// Run sends one command and returns its stdout, or an error for a command the
// jar answered with an exception.
func (j *Jar) Run(args []string, env map[string]string) (string, error) {
	j.nextID++
	line, err := json.Marshal(map[string]interface{}{"id": j.nextID, "args": args, "env": env})
	if err != nil {
		return "", err
	}
	if _, err := j.stdin.Write(append(line, '\n')); err != nil {
		return "", err
	}
	replyLine, err := j.stdout.ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("serve reply: %w", err)
	}
	var reply struct {
		ID     int    `json:"id"`
		OK     bool   `json:"ok"`
		Stdout string `json:"stdout"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(replyLine, &reply); err != nil {
		return "", fmt.Errorf("unreadable serve reply %q: %w", replyLine, err)
	}
	if reply.ID != j.nextID {
		return "", fmt.Errorf("serve reply id %d for request %d", reply.ID, j.nextID)
	}
	if !reply.OK {
		return reply.Stdout, &CommandError{Message: reply.Error}
	}
	return reply.Stdout, nil
}

// CommandError is a command the jar answered with an exception.
type CommandError struct{ Message string }

func (e *CommandError) Error() string { return e.Message }

// ParseResult is what a parse printed: the message's toString, or nothing
// usable (an exception's text, or no message).
type ParseResult struct {
	OK       bool   `json:"ok"`
	ToString string `json:"toString,omitempty"`
}

var messageDump = regexp.MustCompile(`^([A-Za-z0-9]+)\[.*\]$`)

// MessageDump returns the toString cliparser's parse output ends with, if it
// ends with one: the last tab-separated field, as faketandem reads it.
func MessageDump(stdout string) (string, bool) {
	tail := stdout
	if i := strings.LastIndex(stdout, "\t"); i >= 0 {
		tail = stdout[i+1:]
	}
	tail = strings.TrimSpace(tail)
	if !messageDump.MatchString(strings.ReplaceAll(tail, "\n", " ")) {
		return "", false
	}
	return tail, true
}

// JarParse runs one parse case through the jar.
func (j *Jar) JarParse(c ParseCase) (ParseResult, error) {
	env := map[string]string{}
	if c.Characteristic != "" {
		env["PUMPX2_CHARACTERISTIC"] = c.Characteristic
	}
	stdout, err := j.Run([]string{"parse", strings.Join(c.Fragments, " ")}, env)
	if _, ok := err.(*CommandError); ok {
		return ParseResult{}, nil
	}
	if err != nil {
		return ParseResult{}, err
	}
	dump, ok := MessageDump(stdout)
	return ParseResult{OK: ok, ToString: dump}, nil
}

// EncodeResult is what an encode printed.
type EncodeResult struct {
	OK             bool     `json:"ok"`
	MessageName    string   `json:"messageName,omitempty"`
	TxID           string   `json:"txId,omitempty"`
	Packets        []string `json:"packets,omitempty"`
	Characteristic string   `json:"characteristic,omitempty"`
}

// JarEncode runs one encode case through the jar.
func (j *Jar) JarEncode(c EncodeCase) (EncodeResult, error) {
	stdout, err := j.Run([]string{"encode", strconv.Itoa(c.TxID), c.Message, c.Params}, map[string]string{})
	if _, ok := err.(*CommandError); ok {
		return EncodeResult{}, nil
	}
	if err != nil {
		return EncodeResult{}, err
	}
	var printed struct {
		MessageName    string   `json:"messageName"`
		TxID           string   `json:"txId"`
		Packets        []string `json:"packets"`
		Characteristic string   `json:"characteristic"`
	}
	line := strings.TrimSpace(stdout)
	if i := strings.LastIndex(line, "\n"); i >= 0 {
		line = line[i+1:]
	}
	if err := json.Unmarshal([]byte(line), &printed); err != nil {
		// An encode whose constructor threw prints a stack trace and nothing on stdout.
		return EncodeResult{}, nil
	}
	return EncodeResult{OK: true, MessageName: printed.MessageName, TxID: printed.TxID,
		Packets: printed.Packets, Characteristic: printed.Characteristic}, nil
}
