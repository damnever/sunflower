package bufpool

import (
	"bytes"
	"sync"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func Get() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func Put(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}
