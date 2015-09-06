package netio

import (
	"runtime"
)

func StackTrace(all bool) string {
    // Reserve 10K buffer at first
    buf := make([]byte, 10240)

	var size int
    for {
        size = runtime.Stack(buf, all)
        // The size of the buffer may be not enough to hold the stacktrace,
        // so double the buffer size
        if size == len(buf) {
            buf = make([]byte, len(buf)<<1)
            continue
        }
        break
    }

    return string(buf[:size+1])
}