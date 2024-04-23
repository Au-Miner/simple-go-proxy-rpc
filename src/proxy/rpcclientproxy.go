package proxy

import (
	"errors"
	"fmt"
	"github.com/ankur-anand/simple-go-rpc/src/dataserial"
	"github.com/ankur-anand/simple-go-rpc/src/transport"
	"net"
	"reflect"
)

type invocationProxy struct {
	conn net.Conn
}

func NewInvocationProxy(conn net.Conn) invocationProxy {
	return invocationProxy{conn: conn}
}

func (ip invocationProxy) NewProxyInstance(iClass interface{}) {
	rType := reflect.TypeOf(iClass)
	rClass := reflect.ValueOf(iClass).Elem()
	if rType.Kind() != reflect.Ptr {
		panic("Need a pointer of interface struct")
	}
	if rType.Elem().Kind() != reflect.Struct {
		panic("Need a pointer of interface struct")
	}
	rType = rType.Elem()
	for idx := 0; idx < rClass.NumField(); idx++ {
		rElemType := rType.Field(idx)
		rElemClass := rClass.Field(idx)
		if rElemType.Type.Kind() == reflect.Func {
			if !rElemClass.CanSet() {
				continue
			}
			proxyFunc := func(req []reflect.Value) []reflect.Value {
				cReqTransport := transport.NewTransport(ip.conn)
				errorHandler := func(err error) []reflect.Value {
					outArgs := make([]reflect.Value, rElemClass.Type().NumOut())
					for i := 0; i < len(outArgs)-1; i++ {
						outArgs[i] = reflect.Zero(rElemClass.Type().Out(i))
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
				fmt.Println("请求封装的方法名字为", rElemType.Name)
				reqRPC := dataserial.RPCdata{Name: rElemType.Name, Args: inArgs}
				fmt.Println("client准备发送", reqRPC.Name)
				fmt.Println("client准备发送", reqRPC.Args)
				fmt.Println("client准备发送", reqRPC.Err)
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
				fmt.Println("client接收到了", rspDecode.Name)
				fmt.Println("client接收到了", rspDecode.Args)
				fmt.Println("client接收到了", rspDecode.Err)
				if rspDecode.Err != "" { // remote server error
					return errorHandler(errors.New(rspDecode.Err))
				}

				if len(rspDecode.Args) == 0 {
					rspDecode.Args = make([]interface{}, rElemClass.Type().NumOut())
				}
				// unpack response arguments
				// 获取返回值的数量
				numOut := rElemClass.Type().NumOut()
				// 遍历rspDecode(rspRPC)的Args存入outArgs中作为返回值
				outArgs := make([]reflect.Value, numOut)
				for i := 0; i < numOut; i++ {
					if i != numOut-1 { // unpack arguments (except error)
						if rspDecode.Args[i] == nil { // if argument is nil (gob will ignore "Zero" in transmission), set "Zero" value
							// container.Type().Out(i)为返回值的类型
							outArgs[i] = reflect.Zero(rElemClass.Type().Out(i))
						} else {
							outArgs[i] = reflect.ValueOf(rspDecode.Args[i])
						}
					} else { // unpack error argument
						outArgs[i] = reflect.Zero(rElemClass.Type().Out(i))
					}
				}
				return outArgs
			}
			rElemClass.Set(reflect.MakeFunc(rElemClass.Type(), proxyFunc))
		}
	}
}
