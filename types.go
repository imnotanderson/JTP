package JTP

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type Segment struct {
	id   uint32
	data []byte
	addr *net.UDPAddr
}

func (s *Segment) String() string {
	return fmt.Sprintf("[id:%v data:%v]", s.id, s.data)
}

type session struct {
	raddr       net.UDPAddr
	list        []*Segment
	nextId      uint32
	maxId       uint32
	recvBuff    []byte
	mlock       *sync.RWMutex
	ch_readSign chan struct{}
	die         chan struct{}
}

func NewSession(raddr net.UDPAddr) *session {
	return &session{
		raddr:       raddr,
		list:        []*Segment{},
		nextId:      0,
		maxId:       0,
		recvBuff:    []byte{},
		mlock:       new(sync.RWMutex),
		ch_readSign: make(chan struct{}, 1024),
		die:         make(chan struct{}, 1024),
	}
}

func (pSession *session) checkSement(segment *Segment) (ok bool) {
	if segment.id < pSession.nextId {
		return false
	}
	for _, v := range pSession.list {
		if v.id == segment.id {
			return false
		}
	}
	return true
}

func (pSession *session) appendAndSort(segment *Segment) {
	//sort
	insertIdx := len(pSession.list)
	for idx, v := range pSession.list {
		if v.id > segment.id {
			insertIdx = idx
			break
		}
	}
	Log("insertIdx:%v", insertIdx)
	Log("before:%v", pSession.list)
	if insertIdx == len(pSession.list) {
		pSession.list = append(pSession.list, segment)
	} else {
		pSession.list = append(pSession.list[:insertIdx+1], pSession.list[insertIdx:]...)
		pSession.list[insertIdx] = segment
	}
	Log("after:%v", pSession.list)
	if pSession.maxId < segment.id {
		pSession.maxId = segment.id
	}

	for i := 0; i < len(pSession.list); i++ {
		if pSession.nextId == pSession.list[i].id {
			pSession.mlock.Lock()
			pSession.recvBuff = append(pSession.recvBuff, pSession.list[i].data...)
			pSession.mlock.Unlock()
			pSession.list = pSession.list[1:]
			pSession.nextId++
			i--
		} else {
			break
		}
	}
	if len(pSession.recvBuff) > 0 {
		pSession.ch_readSign <- struct{}{}
	}
}

func (s *session) Read(p []byte) (n int, err error) {
	for {
		s.mlock.Lock()
		if len(s.recvBuff) > 0 {
			n = copy(p, s.recvBuff)
			s.recvBuff = s.recvBuff[:0]
			s.mlock.Unlock()
			return n, nil
		}
		s.mlock.Unlock()
		select {
		case <-s.ch_readSign:
		case <-time.After(time.Second):
			return 0, errors.New("timeout")
		}
	}
}

func Log(format string, args ...interface{}) {
	return
	fmt.Printf(format+"\n", args...)
}

type packet struct {
	addr *net.UDPAddr
	data []byte
}
