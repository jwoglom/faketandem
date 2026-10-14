package messages

import (
	"fmt"
	"strings"
)

// JavaException is a failure the jar reports as a thrown exception, named
// for that exception. The parity tests hold only *that* a call fails to the
// jar; the text is for logs.
type JavaException struct {
	Class   string
	Message string
}

func (e *JavaException) Error() string {
	if e.Message == "" {
		return e.Class
	}
	return e.Class + ": " + e.Message
}

func exception(class, format string, args ...interface{}) error {
	return &JavaException{Class: class, Message: fmt.Sprintf(format, args...)}
}

func illegalArgument(format string, args ...interface{}) error {
	return exception("java.lang.IllegalArgumentException", format, args...)
}

// InvalidLength is the error a Parse returns for a payload of the wrong
// size.
func InvalidLength(name string, got, want int) error {
	return illegalArgument("%s: payload of %d bytes, want %d", name, got, want)
}

// Invalid is the error a Parse or constructor returns where pumpX2's
// Validate or Preconditions check throws.
func Invalid(format string, args ...interface{}) error {
	return illegalArgument(format, args...)
}

// recovered turns a panic inside a message's Parse or constructor into an
// error, as Messages.parse catches every exception; a read past the end of
// the payload is pumpX2's ArrayIndexOutOfBoundsException.
func recovered(r interface{}) error {
	if e, ok := r.(error); ok {
		if strings.Contains(e.Error(), "out of range") {
			return exception("java.lang.ArrayIndexOutOfBoundsException", "%v", e)
		}
		return exception("java.lang.RuntimeException", "%v", e)
	}
	return exception("java.lang.RuntimeException", "%v", r)
}
