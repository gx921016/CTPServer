package connect

import (
	"github.com/panjf2000/gnet"
	"go.uber.org/atomic"
	"net"
)

type Client struct {
	Addr    net.Addr
	Uid     *atomic.Int32
	Context *gnet.Conn
	IsReg   *atomic.Bool
}
