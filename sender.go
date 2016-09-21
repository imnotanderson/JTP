package JTP

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type sender_segment struct {
	id   uint32
	data []byte
	//todo:lastSendTime
	sendTime time.Time
}

type reply struct {
	nextId uint32
}

func (ss *sender_segment) getData() []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, ss.id)
	data = append(data, ss.data...)
	return data
}

type Sender struct {
	pConn   *net.UDPConn
	id      uint32
	list    []sender_segment
	ch_send chan []byte
	ch_recv chan reply
	mtu     int
	rto_min time.Duration
	rto_max time.Duration
}

func NewSender(addr string) *Sender {
	pAddr, err := net.ResolveUDPAddr("udp", addr)
	if CheckErr(err) {
		return nil
	}
	pConn, err := net.DialUDP("udp", nil, pAddr)
	if CheckErr(err) {
		return nil
	}
	pSender := &Sender{
		pConn:   pConn,
		list:    []sender_segment{},
		ch_send: make(chan []byte, 1024),
		ch_recv: make(chan reply, 1024),
		mtu:     2,
		rto_min: time.Millisecond * 1,
		rto_max: time.Millisecond * 2,
	}
	go pSender.handleSend()
	go pSender.handleRecv()
	go pSender.handleReply()
	return pSender
}

func (s *Sender) handleSend() {
	for {
		s.send(<-s.ch_send)
	}
}

func (s *Sender) handleReply() {
	for {
		select {
		case replyData := <-s.ch_recv:
			for i := 0; i < len(s.list); i++ {
				if s.list[i].id < replyData.nextId {
					s.list = s.list[i+1:]
					i--
				} else {
					if time.Now().Sub(s.list[i].sendTime) > s.rto_min {
						Log("c.ReSend:%v\n", s.list[i])
						resendCount++
						s.finalSend(s.list[i].getData())
					}
					break
				}
			}
		case <-time.After(s.rto_max):
			if len(s.list) > 0 {
				resendCount++
				s.finalSend(s.list[0].getData())
			}
		}
	}
}

func (c *Sender) handleRecv() {
	for {
		data := make([]byte, 1024)
		l, err := c.pConn.Read(data)
		if CheckErr(err) {
			continue
		}
		nextId, _, ok := c.parseRecvData(data[:l])
		if ok {
			Log("c.recv.nextId:%v\n", nextId)
			c.ch_recv <- reply{nextId: nextId}
		}
	}
}

func (c *Sender) parseRecvData(data []byte) (nextId uint32, max uint32, ok bool) {
	if len(data) < 8 {
		ok = false
		return
	}
	ok = true
	nextId = binary.LittleEndian.Uint32(data[:4])
	max = binary.LittleEndian.Uint32(data[4:8])
	return
}

func (c *Sender) Send(data []byte) {
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

func (s *Sender) send(data []byte) {
	segment := sender_segment{
		id:       s.id,
		data:     data,
		sendTime: time.Now(),
	}
	Log("c.send:id:%v,data:%v", segment.id, segment.data)
	s.id++
	s.list = append(s.list, segment)
	s.finalSend(segment.getData())
}

var sendCount = 0
var resendCount = 0

func (s *Sender) finalSend(data []byte) {
	sendCount += len(data)
	fmt.Printf("sendCount:%v resendCount:%v\n", sendCount, resendCount)
	s.pConn.Write(data)
}
