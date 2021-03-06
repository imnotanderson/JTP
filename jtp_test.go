package JTP

import (
	"fmt"
	"testing"
	"time"
)

func TestJTP(t *testing.T) {
	go S()
	go C()
	<-time.After(time.Second)
	C()
}

var allSendData []byte

func S() {
	allSendData = []byte{}
	s := NewSender("127.0.0.1:12345")
	//for i := 0; i < 100; i++ {
	data := []byte(fmt.Sprint("hello world"))
	<-time.After(time.Millisecond * 300)
	allSendData = append(allSendData, data...)
	s.Send(data)
	//	}
}

func C() {
	r := NewReceiver()
	r.Dial("127.0.0.1:12345")
	allRecvData := []byte{}
	for {
		data := make([]byte, 1024)
		l, err := r.Read(data)
		if CheckErr(err) {
			return
		}
		allRecvData = append(allRecvData, data[:l]...)
		fmt.Printf("-->%v\n", allRecvData)
		fmt.Printf("-->%v\n", allSendData)
	}
}
