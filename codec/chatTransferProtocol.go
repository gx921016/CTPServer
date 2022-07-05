package codec

import (
	"CTPServer/connect"
	"CTPServer/global"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/panjf2000/gnet"
	"go.uber.org/atomic"
)

type ChatTransferProtocol struct {
	UID     int32
	CtpType int8
}



func (cc *ChatTransferProtocol) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	result := make([]byte, 0)

	buffer := bytes.NewBuffer(result)

	if err := binary.Write(buffer, binary.BigEndian, int8(cc.CtpType)); err != nil {
		s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil, errors.New(s)
	}
	// take out the param
	dataLen := int32(len(buf))
	if err := binary.Write(buffer, binary.BigEndian, dataLen); err != nil {
		s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil, errors.New(s)
	}
	if err := binary.Write(buffer, binary.BigEndian, cc.UID); err != nil {
		s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil, errors.New(s)
	}
	if dataLen > 0 {
		if err := binary.Write(buffer, binary.BigEndian, buf); err != nil {
			s := fmt.Sprintf("Pack data error , %v", err)
			return nil, errors.New(s)
		}
	}

	return buffer.Bytes(), nil
}
func (cc *ChatTransferProtocol) Decode1(c []byte) ([]byte, error) {
	return nil, nil
}

// Decode ...
func (cc *ChatTransferProtocol) Decode(c gnet.Conn) ([]byte, error) {

	if verifyCTP(c) {
		return nil, errors.New("协议错误")
	}
	// parse header
	_, buf := c.ReadN(DefaultHeadLength)
	byteBuffer := bytes.NewBuffer(buf)
	var tag int8
	_ = binary.Read(byteBuffer, binary.BigEndian, &tag)
	switch tag {
	case MSG:
		var dataLength int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &dataLength)
		dataLen := int(dataLength) // max int32 can contain 210MB payload
		protocolLen := MsgHeadLength + dataLen
		if dataSize, data := c.ReadN(protocolLen); dataSize == protocolLen {
			c.ResetBuffer()
			return data[DefaultHeadLength-4:], nil
		}

	case REGISTERED:
		var dataLength int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &dataLength)

		var uid int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &uid)
		protocolLen := DefaultHeadLength + int(dataLength)
		if dataSize, data := c.ReadN(protocolLen); dataSize == protocolLen {
			c.ShiftN(protocolLen)
			c.ResetBuffer()
			token := data[DefaultHeadLength:]
			//验证注册合法性
			if !verifyRegister(string(token), uid) {
				buffer := bytes.NewBufferString("用户验证失败")
				cc.CtpType = ERROR
				cc.UID = uid
				_ = c.AsyncWrite(buffer.Bytes())
				c.Close()
				fmt.Printf("用户验证失败,%s uid = %d\n", string(token), uid)
				return nil, errors.New("用户验证失败")
			}
			if store, loaded := global.ClientsMap.Load(c.RemoteAddr()); loaded {
				client := store.(*connect.Client)
				if client.IsReg.Load() == false {
					client.IsReg.Store(true)
				}
				if client.Uid.Load() == 0 {
					client.Uid.Store(uid)
				}
			} else {
				client := &connect.Client{
					Addr:    c.RemoteAddr(),
					Uid:     atomic.NewInt32(uid),
					Context: &c,
					IsReg:   atomic.NewBool(true),
				}
				global.ClientsMap.Store(c.RemoteAddr(), client)
			}
			//验证注册成功存储到在线map中
			global.OnlineClientMap.Store(uid, &c)
			store, _ := global.ClientsMap.Load(c.RemoteAddr())
			fmt.Printf("验证成功 %+v\n", store)
			getAliyunToken()
			buffer := bytes.NewBufferString("ok")
			cc.UID = uid
			cc.CtpType = ACK
			err := c.AsyncWrite(buffer.Bytes())
			return nil, err
		}

	case HEARBEAT:
		//fmt.Println("心跳")
		_, data := c.ReadN(1)
		//c.ResetBuffer()
		c.ShiftN(c.BufferLength())
		return data, nil
	case ACK:
		//fmt.Println("ACK")
		_, data := c.ReadN(c.BufferLength())
		c.ShiftN(c.BufferLength())
		return data[DefaultHeadLength-4:], nil

	default:
		_, data := c.ReadN(c.BufferLength())
		c.ShiftN(c.BufferLength())
		//fmt.Println("没找到")
		return data[DefaultHeadLength-4:], nil

	}
	// log.Println("not enough header data:", size)
	return nil, errors.New("not enough header data")
}

func verifyCTP(c gnet.Conn) bool {
	if c.BufferLength() > DefaultHeadLength {
		return true
	}
	return false
}
func verifyRegister(token string, uid int32) bool {
	if string(token) != "qwertyu" || uid == 0 {
		return false
	}
	return true
}
