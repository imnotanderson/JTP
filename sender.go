package JTP

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type sender_segment struct {
	id       uint32
	data     []byte
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

type conn_state int

type Sender struct {
	allData  []byte
	mMapLock *sync.RWMutex
	connMap  map[string]*JTPConn
	pConn    *net.UDPConn
	ch_conn  chan *JTPConn
	ch_send  chan packet
}

func NewSender(str_laddr string) *Sender {
	lAddr, err := net.ResolveUDPAddr("udp", str_laddr)
	if CheckErr(err) {
		return nil
	}
	pConn, err := net.ListenUDP("udp", lAddr)
	if CheckErr(err) {
		return nil
	}
	pSender := &Sender{
		pConn:    pConn,
		connMap:  make(map[string]*JTPConn),
		ch_conn:  make(chan *JTPConn, 1024),
		ch_send:  make(chan packet, 1024),
		mMapLock: new(sync.RWMutex),
	}
	go pSender.raw_recv()
	go pSender.raw_send()
	return pSender
}

func (s *Sender) raw_send() {
	for {
		pkt := <-s.ch_send
		_, err := s.pConn.WriteTo(pkt.data, pkt.addr)
		CheckErr(err)
	}
}

func (s *Sender) raw_recv() {
	Log("recv begin")
	for {
		data := make([]byte, 1024)
		l, addr, err := s.pConn.ReadFromUDP(data)
		if CheckErr(err) {
			continue
		}
		s.mMapLock.RLock()
		conn := s.connMap[addr.String()]
		s.mMapLock.RUnlock()
		if conn == nil {
			conn = buildJTPConn(addr, s.ch_send)
			s.mMapLock.Lock()
			conn.send(s.allData)
			s.connMap[addr.String()] = conn
			Log("new conn join")
			s.mMapLock.Unlock()
		}
		conn.handleRecv(data[:l], s.ch_conn)
	}
}

func (s *Sender) Send(data []byte) {
	s.mMapLock.RLock()
	s.allData = append(s.allData, data...)
	for _, v := range s.connMap {
		v.send(data)
	}
	s.mMapLock.RUnlock()
}
