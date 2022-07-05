package main

import (
	codec2 "CTPServer/codec"
	"CTPServer/connect"
	"CTPServer/global"
	"bytes"
	"flag"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"go.uber.org/atomic"
	"log"
)

type ctpServer struct {
	*gnet.EventServer
	addr       string
	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func main() {

	var port int
	var multicore, reuseport bool

	// Example command: go run echo.go --port 9000 --multicore=true --reuseport=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", true, "--multicore true")
	flag.BoolVar(&reuseport, "reuseport", true, "--reuseport true")
	flag.Parse()
	codec := &codec2.ChatTransferProtocol{UID: 0}
	echo := &ctpServer{
		addr:       fmt.Sprintf("tcp://:%d", port),
		codec:      codec,
		workerPool: goroutine.Default(),
	}
	_ = gnet.Serve(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport)) //,gnet.WithCodec(codec)
	//log.Fatal(gnet.Serve(echo, fmt.Sprintf("udp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport), gnet.WithCodec(codec)))
}

func (es *ctpServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("UDP Echo server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (es *ctpServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {

	fmt.Printf("%s进入了\n", c.RemoteAddr().String())
	client := &connect.Client{
		Addr:    c.RemoteAddr(),
		Uid:     atomic.NewInt32(0),
		Context: &c,
		IsReg:   atomic.NewBool(false),
	}
	global.ClientsMap.Store(c.RemoteAddr(), client)
	//es.clientsCoMap.Set(concurrent_map.StrKey(c.RemoteAddr().String()),out)
	return
}

func (es *ctpServer) OnShutdown(svr gnet.Server) {
}
func (es *ctpServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	global.ClientsMap.Delete(c.RemoteAddr())
	fmt.Printf("%s滚蛋了\n", c.RemoteAddr().String())
	return
}

func (es *ctpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	data := append([]byte{}, frame...)
	//fmt.Printf("ordata = %s \n",string(data))
	//item :=codec2.ChatTransferProtocol{
	//	UID:        0,
	//
	//}
	//c.SetContext()

	//item := codec2.ChatTransferProtocol{
	//	UID:    to,
	//	CtpType: codec2.MSG,
	//}
	//c.SetContext(item)
	_ = es.workerPool.Submit(func() {
		buffer := bytes.NewBuffer(data)
		msg, _ := codec2.Decode1(&c, buffer)
		if msg != nil {
			fmt.Printf("%d 发送给 %d 内容为：%s \n", msg.From, msg.To, msg.Palyload)
			if value, ok := global.OnlineClientMap.Load(msg.To); ok {
				con := value.(*gnet.Conn)
				fmt.Println((*con).RemoteAddr())
				palyload := bytes.NewBufferString(msg.Palyload)
				(*con).AsyncWrite(codec2.Encode1(codec2.MSG, msg.From, palyload.Bytes()))
			}
		}
		//global.clientsMap.Range(func(key, value interface{}) bool {
		//codec:= es.codec.(*codec2.ChatTransferProtocol)
		//codec.UID = uid
		//codec.CtpType = codec2.MSG
		//fmt.Printf("%d to %d : %s \n", uid, to, buffer)
		//if value, ok := global.OnlineClientMap.Load(to); ok {
		//	con := value.(*gnet.Conn)
		//	fmt.Println((*con).RemoteAddr())
		//	(*con).AsyncWrite(buffer.Bytes())
		//}

		//	//if key != c.RemoteAddr() {
		//	//	con := value.(gnet.Conn)
		//	//	con.AsyncWrite(data)
		//	//}
		//	return true
		//})
	})

	return
}
