package client

import (
	"errors"
	"net"
	"reflect"

	"github.com/ankur-anand/simple-go-rpc/src/dataserial"
	"github.com/ankur-anand/simple-go-rpc/src/transport"
)

// Client struct
type Client struct {
	conn net.Conn
}

// NewClient creates a new client
func NewClient(conn net.Conn) *Client {
	return &Client{conn}
}

// CallRPC Method
func (c *Client) CallRPC(rpcName string, fPtr interface{}) {
	// 如果reflect.Value.kind()为reflect.Ptr类型，则.Elem()方法返回指针指向的变量
	// 如果reflect.Value.kind()为reflect.Interface类型，则.Elem()方法用于获取接口对应的动态值
	// 我们可以通过调用reflect.ValueOf(&x).Elem()，来获取任意变量x对应的可取地址的Value
	// 在这里对应着接口对应的执行方法
	container := reflect.ValueOf(fPtr).Elem()
	f := func(req []reflect.Value) []reflect.Value {
		cReqTransport := transport.NewTransport(c.conn)
		errorHandler := func(err error) []reflect.Value {
			outArgs := make([]reflect.Value, container.Type().NumOut())
			for i := 0; i < len(outArgs)-1; i++ {
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
			outArgs[len(outArgs)-1] = reflect.ValueOf(&err).Elem()
			return outArgs
		}

		// Process input parameters
		inArgs := make([]interface{}, 0, len(req))
		for _, arg := range req {
			inArgs = append(inArgs, arg.Interface())
		}

		// ReqRPC
		reqRPC := dataserial.RPCdata{Name: rpcName, Args: inArgs}
		b, err := dataserial.Encode(reqRPC)
		if err != nil {
			panic(err)
		}
		err = cReqTransport.Send(b)
		if err != nil {
			return errorHandler(err)
		}
		// receive response from server
		rsp, err := cReqTransport.Read()
		if err != nil { // local network error or decode error
			return errorHandler(err)
		}
		rspDecode, _ := dataserial.Decode(rsp)
		if rspDecode.Err != "" { // remote server error
			return errorHandler(errors.New(rspDecode.Err))
		}

		if len(rspDecode.Args) == 0 {
			rspDecode.Args = make([]interface{}, container.Type().NumOut())
		}
		// 获取返回值的数量
		numOut := container.Type().NumOut()
		// 遍历rspDecode(rspRPC)的Args存入outArgs中作为返回值
		outArgs := make([]reflect.Value, numOut)
		for i := 0; i < numOut; i++ {
			if i != numOut-1 {
				if rspDecode.Args[i] == nil {
					// container.Type().Out(i)为返回值的类型
					outArgs[i] = reflect.Zero(container.Type().Out(i))
				} else {
					outArgs[i] = reflect.ValueOf(rspDecode.Args[i])
				}
			} else {
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
		}
		return outArgs
	}
	container.Set(reflect.MakeFunc(container.Type(), f))
}
