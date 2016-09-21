package JTP

import (
	"encoding/binary"
	"net"
)

type Receiver struct {
	pConn      *net.UDPConn
	sessionMap map[string]*session
	ch_session chan *session
}

func NewReceiver() *Receiver {
	return &Receiver{
		sessionMap: make(map[string]*session),
		ch_session: make(chan *session, 1024),
	}
}
func (r Receiver) log(str string, args ...interface{}) {
	Log(str+"\n", args...)
}

func (r *Receiver) Listen(addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if CheckErr(err) {
		return
	}
	r.log("listen " + addr)
	r.pConn, err = net.ListenUDP("udp", udpAddr)
	if CheckErr(err) {
		return
	}
	go r.recv()
}

func (r *Receiver) Accept() *session {
	return <-r.ch_session
}

func (r *Receiver) recv() {
	print("svr linten start")
	defer func() { Log("recv exit") }()
	for {
		data := make([]byte, 1024)
		l, addr, err := r.pConn.ReadFromUDP(data)
		if CheckErr(err) {
			continue
		}

		//fixme:模拟掉包
		debug := debugMissing[debugIdx%len(debugMissing)]
		debugIdx++
		if debug > 0 {
			continue
		}

		if l < 4 {
			continue
		}
		id := binary.LittleEndian.Uint32(data)
		segment := &Segment{
			id:   id,
			data: data[4:l],
			addr: addr,
		}
		//get session
		pSession := r.getSession(segment)
		//check
		if pSession.checkSement(segment) == false {
			continue
		}

		r.appendSegment(segment)
		r.reply(segment)
	}
}

func (r *Receiver) getSession(segment *Segment) *session {
	pSession := r.sessionMap[segment.addr.String()]
	if pSession == nil {
		pSession = NewSession(*segment.addr)
		select {
		case r.ch_session <- pSession:
			r.sessionMap[segment.addr.String()] = pSession
		default:
			//s.log("ch_session full")
			panic("ch_session full")
		}
	}
	return pSession
}

//>0掉包
var debugIdx = 0
var debugMissing = []int{0}

//reply first missing segment & max segment id
func (r *Receiver) reply(segment *Segment) {
	pSession := r.sessionMap[segment.addr.String()]
	if pSession == nil {
		Log("reply session nil")
		return
	}
	data := r.getReplyData(pSession.nextId, pSession.maxId)
	Log("s.sendReply nextId:%v maxId:%v", pSession.nextId, pSession.maxId)
	r.pConn.WriteToUDP(data, &pSession.addr)
}

func (r *Receiver) getReplyData(nextId uint32, maxId uint32) []byte {
	dataNext := make([]byte, 4)
	dataMax := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataNext, nextId)
	binary.LittleEndian.PutUint32(dataMax, maxId)
	return append(dataNext, dataMax...)
}

//sort append by segment.id  ,append continuous data 2 recv_buf
func (r *Receiver) appendSegment(segment *Segment) {
	r.log("append segment:%v", segment)
	pSession := r.sessionMap[segment.addr.String()]
	if pSession == nil {
		pSession = NewSession(*segment.addr)
		select {
		case r.ch_session <- pSession:
			r.sessionMap[segment.addr.String()] = pSession
		default:
			r.log("ch_session full")
			return
		}
	}
	pSession.appendAndSort(segment)
	r.log("recv data :%v", pSession.recvBuff)
	r.log("sort:%v", pSession.list)
}

func CheckErr(err error) bool {
	if err != nil {
		Log("err:%v", err.Error())
		return true
	}
	return false
}
