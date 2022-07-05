package codec

import (
	"CTPServer/connect"
	"CTPServer/global"
	"CTPServer/model"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/panjf2000/gnet"
	"go.uber.org/atomic"
)

const (
	DefaultHeadLength      = 9
	MsgHeadLength          = 13
	DefaultProtocolVersion = 0x8001 // test protocol version
	MSG                    = 1
	HEARBEAT               = 2
	REGISTERED             = 3
	ACK                    = 4
	ERROR                  = 0x000F
)
func  getAliyunToken()  {
	client, err := sts.NewClientWithAccessKey("cn-beijing", "LTAI5t8STcMiyAPQddBQqJAL", "0MnpNgvtj93CDQltEAUgfWMQbspVWn")
	//构建请求对象。
	request := sts.CreateAssumeRoleRequest()
	request.Scheme = "https"
	//设置参数。关于参数含义和设置方法，请参见《API参考》。
	request.RoleArn = "acs:ram::261940522450858400:role/adminrole"
	request.RoleSessionName = "gao"

	
	//发起请求，并得到响应。
	response, err := client.AssumeRole(request)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Printf("response is %#v\n", response)
}
func Decode1(c *gnet.Conn, byteBuffer *bytes.Buffer) (*model.Message, error) {
	var tag int8
	_ = binary.Read(byteBuffer, binary.BigEndian, &tag)
	switch tag {
	case MSG:
		var dataLength int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &dataLength)
		var to int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &to)
		var uid int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &uid)
		playload:=byteBuffer.Bytes()
		msg:=&model.Message{
			Palyload: string(playload),
			To:       to,
			From:     uid,
			Type:     0,
		}
		return msg,nil
	case HEARBEAT:
		//println("beat")
		return nil,nil
	case REGISTERED:
		var dataLength int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &dataLength)

		var uid int32
		_ = binary.Read(byteBuffer, binary.BigEndian, &uid)
		//protocolLen := DefaultHeadLength + int(dataLength)
		//验证注册合法性
		var token []byte
		token = byteBuffer.Bytes()
		if !verifyRegister(string(token), uid) {
			buffer := bytes.NewBufferString("用户验证失败")
			_ = (*c).AsyncWrite(Encode1(ACK, uid, buffer.Bytes()))
			//_ = (*c).AsyncWrite(buffer.Bytes())
			(*c).Close()
			fmt.Printf("用户验证失败,%s uid = %d\n", string(token), uid)
			return nil, errors.New("用户验证失败")
		}
		if store, loaded := global.ClientsMap.Load((*c).RemoteAddr()); loaded {
			client := store.(*connect.Client)
			if client.IsReg.Load() == false {
				client.IsReg.Store(true)
			}
			if client.Uid.Load() == 0 {
				client.Uid.Store(uid)
			}
		} else {
			client := &connect.Client{
				Addr:    (*c).RemoteAddr(),
				Uid:     atomic.NewInt32(uid),
				Context: c,
				IsReg:   atomic.NewBool(true),
			}
			global.ClientsMap.Store((*c).RemoteAddr(), client)
		}
		//验证注册成功存储到在线map中
		global.OnlineClientMap.Store(uid, c)
		store, _ := global.ClientsMap.Load((*c).RemoteAddr())
		fmt.Printf("验证成功 %+v\n", store)
		getAliyunToken()
		buffer := bytes.NewBufferString("ok")

		err := (*c).AsyncWrite(Encode1(ACK, uid, buffer.Bytes()))
		return nil, err

	case ACK:
	}
	return nil, nil
}

func Encode1(ctpType int8, uid  int32, buf []byte) []byte {
	result := make([]byte, 0)

	buffer := bytes.NewBuffer(result)

	if err := binary.Write(buffer, binary.BigEndian, ctpType); err != nil {
		//s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil
	}
	// take out the param
	dataLen := int32(len(buf))
	if err := binary.Write(buffer, binary.BigEndian, dataLen); err != nil {
		//s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil
	}
	if err := binary.Write(buffer, binary.BigEndian, uid); err != nil {
		//s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil
	}
	if dataLen > 0 {
		if err := binary.Write(buffer, binary.BigEndian, buf); err != nil {
			//s := fmt.Sprintf("Pack data error , %v", err)
			return nil
		}
	}
	return buffer.Bytes()
}
