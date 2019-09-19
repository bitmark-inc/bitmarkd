package p2p

import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

//basicStream for Adding more infomation for handleStream
type basicStream struct {
	ID string
}

func (s *basicStream) handleStream(stream network.Stream) {
	log.Println("--- Start A New stream --")
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw, s.ID)
	go writeData(rw, s.ID)

	// stream 's' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter, who string) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("%s RECIEVE:\x1b[32m%s\x1b[0m> ", who, str)
		}
	}
}

func writeData(rw *bufio.ReadWriter, who string) {
	timeUp := time.After(5 * time.Second)
	for {
		select {
		case <-timeUp:
			rw.WriteString(fmt.Sprintf("%s SEND %s\n", who, time.Now().UTC().String()))
			rw.Flush()
		}
	}
}
