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

func GrowGet(n int) *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Grow(n)
	return buf
}

func Put(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}
