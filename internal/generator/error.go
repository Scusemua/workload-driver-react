package generator

import "fmt"

type Error struct {
	error

	msg string
}

func Errorf(err error, msg string, args ...interface{}) error {
	return &Error{
		error: err,
		msg:   fmt.Sprintf("%v: %s", err, fmt.Sprintf(msg, args...)),
	}
}

func IsError(err1 error, err2 error) bool {
	if err1 == err2 {
		return true
	} else if e, ok := err1.(*Error); ok {
		return e.error == err2
	} else if e, ok := err2.(*Error); ok {
		return err1 == e.error
	} else {
		return false
	}
}

func (err *Error) Error() string {
	return err.msg
}
