package JTP

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type JTPConn struct {
	addr           *net.UDPAddr
	id             uint32
	list           []sender_segment
	ch_send        chan []byte
	ch_recv        chan reply
	mtu            int
	rto_min        time.Duration
	rto_max        time.Duration
	die            chan struct{}
	ch_raw_send    chan<- packet
	maxResendCount int
	resendCount    int
}

func buildJTPConn(addr *net.UDPAddr, ch_raw_send chan<- packet) *JTPConn {
	conn := &JTPConn{
		addr:           addr,
		list:           []sender_segment{},
		ch_send:        make(chan []byte, 1024),
		ch_recv:        make(chan reply, 1024),
		mtu:            2,
		rto_min:        time.Millisecond * 1,
		rto_max:        time.Millisecond * 2,
		die:            make(chan struct{}),
		ch_raw_send:    ch_raw_send,
		maxResendCount: 10,
	}
	go conn.handleReply()
	go conn.handleSend()
	return conn
}

func (s *JTPConn) handleSend() {
	for {
		select {
		case data := <-s.ch_send:
			s.send(data)
		case <-s.die:
			return
		}

	}
}

func (s *JTPConn) handleReply() {
	for {
		select {
		case replyData := <-s.ch_recv:
			s.resendCount = 0
			for i := 0; i < len(s.list); i++ {
				if s.list[i].id < replyData.nextId {
					s.list = s.list[i+1:]
					i--
				} else {
					if time.Now().Sub(s.list[i].sendTime) > s.rto_min {
						Log("conn.resend:%v\n", s.list[i])
						resendCount++
						s.raw_send(s.list[i].getData())
					}
					break
				}
			}
		case <-time.After(s.rto_max):
			if len(s.list) > 0 {
				resendCount++
				s.raw_send(s.list[0].getData())
				//todo:max_resebdCount
				s.resendCount++
				if s.resendCount > s.maxResendCount {
					close(s.die)
				}
			}
		case <-s.die:
			return
		}
	}
}

func (conn *JTPConn) handleRecv(data []byte, ch_conn chan<- *JTPConn) {
	nextId, _, ok := conn.parseRecvData(data)
	if ok {
		Log("c.recv.nextId:%v\n", nextId)
		conn.ch_recv <- reply{nextId: nextId}
	}
	//todo:if recv close sgin close(die)
}

func (c *JTPConn) parseRecvData(data []byte) (nextId uint32, max uint32, ok bool) {
	if len(data) < 8 {
		ok = false
		return
	}
	ok = true
	nextId = binary.LittleEndian.Uint32(data[:4])
	max = binary.LittleEndian.Uint32(data[4:8])
	return
}

func (c *JTPConn) Send(data []byte) {
	Log("send:%v\n", data)
	for _, v := range splitData(data, c.mtu) {
		c.ch_send <- v
	}
}

//split data by mtu
func splitData(data []byte, mtu int) [][]byte {
	if mtu <= 0 {
		panic("mtu <= 0")
		return nil
	}
	result := [][]byte{}
	for len(data) > mtu {
		newData := make([]byte, mtu)
		copy(newData, data[:mtu])
		result = append(result, newData)
		data = data[mtu:]
	}
	if len(data) > 0 {
		result = append(result, data)
	}
	return result
}

func (s *JTPConn) send(data []byte) {
	segment := sender_segment{
		id:       s.id,
		data:     data,
		sendTime: time.Now(),
	}
	Log("s.send:id:%v,data:%v", segment.id, segment.data)
	s.id++
	s.list = append(s.list, segment)
	s.raw_send(segment.getData())
}

var sendCount = 0
var resendCount = 0

func (s *JTPConn) raw_send(data []byte) {
	sendCount += len(data)
	fmt.Printf("sendCount:%v resendCount:%v\n", sendCount, resendCount)
	s.ch_raw_send <- packet{
		addr: s.addr,
		data: data,
	}
}
