package conn

import (
	"io"
	"net"
	"sync"

	"github.com/damnever/sunflower/pkg/bufpool"
)

// LinkStream links two connection,
// it make one end read and write another end, vice versa.
func LinkStream(inConn, outConn net.Conn) (int64, int64) {
	wg := sync.WaitGroup{}
	var inBytes, outBytes int64

	copyStream := func(out, in net.Conn, traffic *int64) {
		defer wg.Done()
		defer out.Close()

		buf := bufpool.GrowGet(2048)
		defer bufpool.Put(buf)
		*traffic, _ = io.CopyBuffer(out, in, buf.Bytes()[:2048])
		if tc, ok := out.(*net.TCPConn); ok {
			tc.CloseRead()
			tc.CloseWrite()
		}
	}

	wg.Add(2)
	go copyStream(outConn, inConn, &inBytes)
	go copyStream(inConn, outConn, &outBytes)
	wg.Wait()
	return inBytes, outBytes
}
