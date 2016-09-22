package JTP

import (
	"encoding/binary"
	"net"
)

type Receiver struct {
	pConn    *net.UDPConn
	pSession *session
}

func NewReceiver() *Receiver {
	return &Receiver{}
}
func (r Receiver) log(str string, args ...interface{}) {
	Log("Receiver:"+str+"\n", args...)
}

func (r *Receiver) Dial(str_raddr string) error {
	raddr, err := net.ResolveUDPAddr("udp", str_raddr)
	if CheckErr(err) {
		return err
	}
	udpConn, err := net.DialUDP("udp", nil, raddr)
	if CheckErr(err) {
		return err
	}
	r.pConn = udpConn
	r.pSession = NewSession(*raddr)
	go r.recv()
	data := r.getReplyData(0, 0)
	_, err = r.pConn.Write(data)
	if CheckErr(err) {
		return err
	}
	return nil
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
		if pSession == nil {
			continue
		}
		//check
		if pSession.checkSement(segment) == false {
			continue
		}
		r.appendSegment(segment)
		r.reply(segment)
	}
}

func (r *Receiver) getSession(segment *Segment) *session {

	if r.pSession == nil {
		r.pSession = NewSession(*segment.addr)
		return r.pSession
	}
	if r.pSession.raddr.String() != segment.addr.String() {
		return nil
	}
	return r.pSession
}

//>0掉包
var debugIdx = 0
var debugMissing = []int{0, 1}

//reply first missing segment & max segment id
func (r *Receiver) reply(segment *Segment) {
	pSession := r.pSession
	if pSession == nil {
		Log("reply session nil")
		return
	}
	data := r.getReplyData(pSession.nextId, pSession.maxId)
	Log("s.sendReply nextId:%v maxId:%v", pSession.nextId, pSession.maxId)
	if pSession.raddr.String() != r.pConn.RemoteAddr().String() {
		Log("pSession.raddr.String()!=r.pConn.RemoteAddr() return")
		return
	}
	_, err := r.pConn.Write(data)
	if CheckErr(err) {
		panic(err)
	}
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
	pSession := r.pSession
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

func (r *Receiver) Read(p []byte) (l int, err error) {
	return r.pSession.Read(p)
}
