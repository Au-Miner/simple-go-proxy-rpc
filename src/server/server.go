package server

import (
	"fmt"
	"github.com/ankur-anand/simple-go-rpc/src/dataserial"
	"github.com/ankur-anand/simple-go-rpc/src/transport"
	"io"
	"log"
	"net"
	"reflect"
)

// RPCServer ...
type RPCServer struct {
	addr  string
	funcs map[string]reflect.Value
}

// NewServer creates a new server
func NewServer(addr string) *RPCServer {
	return &RPCServer{addr: addr, funcs: make(map[string]reflect.Value)}
}

func (s *RPCServer) Register(iClass interface{}) {
	rType := reflect.TypeOf(iClass)
	rClass := reflect.ValueOf(iClass)
	fmt.Println("rClasses中的方法个数为：", rClass.NumMethod())
	for idx := 0; idx < rClass.NumMethod(); idx++ {
		rFuncType := rType.Method(idx)
		rFuncClass := rClass.Method(idx)
		if rFuncType.Type.Kind() == reflect.Func {
			fmt.Println("rClass.CanSet()：", rFuncClass.CanSet())
			fmt.Println("准备注册rType.Name为: ", rFuncType.Name)
			if _, ok := s.funcs[rFuncType.Name]; ok {
				continue
			}
			s.funcs[rFuncType.Name] = rFuncClass
		}
	}
}

// Execute the given function if present
func (s *RPCServer) Execute(req dataserial.RPCdata) dataserial.RPCdata {
	fmt.Println("server接收到了", req.Name)
	fmt.Println("server接收到了", req.Args)
	fmt.Println("server接收到了", req.Err)
	// get method by name
	f, ok := s.funcs[req.Name]
	if !ok {
		// since method is not present
		e := fmt.Sprintf("func %s not Registered", req.Name)
		log.Println(e)
		return dataserial.RPCdata{Name: req.Name, Args: nil, Err: e}
	}

	log.Printf("func %s is called\n", req.Name)
	// unpack request arguments
	inArgs := make([]reflect.Value, len(req.Args))
	for i := range req.Args {
		inArgs[i] = reflect.ValueOf(req.Args[i])
	}

	// invoke requested method
	out := f.Call(inArgs)
	fmt.Println("server的out为", out)
	resArgs := make([]interface{}, len(out)-1)
	for i := 0; i < len(out)-1; i++ {
		// 将前len(out)-1个返回值转换为interface{}类型
		resArgs[i] = out[i].Interface()
	}
	// pack error argument
	var er string
	if _, ok := out[len(out)-1].Interface().(error); ok {
		// convert the error into error string value
		er = out[len(out)-1].Interface().(error).Error()
	}
	return dataserial.RPCdata{Name: req.Name, Args: resArgs, Err: er}
}

// Run server
func (s *RPCServer) Run() {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("listen on %s err: %v\n", s.addr, err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept err: %v\n", err)
			continue
		}
		go func() {
			connTransport := transport.NewTransport(conn)
			for {
				// read request
				req, err := connTransport.Read()
				if err != nil {
					if err != io.EOF {
						log.Printf("read err: %v\n", err)
						return
					}
				}
				// decode the data and pass it to execute
				decReq, err := dataserial.Decode(req)
				if err != nil {
					log.Printf("Error Decoding the Payload err: %v\n", err)
					return
				}
				// get the executed result.
				resP := s.Execute(decReq)
				// encode the data back
				b, err := dataserial.Encode(resP)
				if err != nil {
					log.Printf("Error Encoding the Payload for response err: %v\n", err)
					return
				}
				// send response to client
				err = connTransport.Send(b)
				if err != nil {
					log.Printf("transport write err: %v\n", err)
				}
			}
		}()
	}
}
