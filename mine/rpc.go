// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file in this directory.

// This is from go src net/rpc/server.go
//
// It is stripped down to provide only the server registration system
// so as to allow compatability with the Go library RPC callback API.

/*
	Package rpc provides access to the exported methods of an object across a
	network or other I/O connection.  A server registers an object, making it visible
	as a service with the name of the type of the object.  After registration, exported
	methods of the object will be accessible remotely.  A server may register multiple
	objects (services) of different types but it is an error to register multiple
	objects of the same type.

	Only methods that satisfy these criteria will be made available for remote access;
	other methods will be ignored:

		- the method is exported.
		- the method has two arguments, both exported (or builtin) types.
		- the method's second argument is a pointer.
		- the method has return type error.

	In effect, the method must look schematically like

		func (t *T) MethodName(argType T1, replyType *T2) error

	where T, T1 and T2 can be marshaled by encoding/gob.
	These requirements apply even if a different codec is used.
	(In the future, these requirements may soften for custom codecs.)

	The method's first argument represents the arguments provided by the caller; the
	second argument represents the result parameters to be returned to the caller.
	The method's return value, if non-nil, is passed back as a string that the client
	sees as if created by errors.New.  If an error is returned, the reply parameter
	will not be sent back to the client.

	Here is a simple example.  A server wishes to export an object of type Arith:

	Note: arg:"N" annotation is required because this RPC
              parameter values are passsed as an array and must be
              extracted from this array into the struct.  Not needed for
	      the result type as this will be directly converted to JSON.

		package server

		type Args struct {
			A int `arg:"0"`
			B int `arg:"1"`
		}

		type Quotient struct {
			Quo, Rem int
		}

		type Arith int

		func (t *Arith) Multiply(args *Args, reply *int) error {
			*reply = args.A * args.B
			return nil
		}

		func (t *Arith) Divide(args *Args, quo *Quotient) error {
			if args.B == 0 {
				return errors.New("divide by zero")
			}
			quo.Quo = args.A / args.B
			quo.Rem = args.A % args.B
			return nil
		}

	The server calls:

		arith := new(Arith)
		rpc.Register(arith)
*/
package mine

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// Server represents an RPC Server.
type Server struct {
	mu         sync.RWMutex // protects the serviceMap
	serviceMap map[string]*service
}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{serviceMap: make(map[string]*service)}
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//	- exported method
//	- two arguments, both of exported type
//	- the second argument is a pointer
//	- one return value, of type error
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *Server) Register(rcvr interface{}) error {
	return server.register(rcvr, "", false)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (server *Server) RegisterName(name string, rcvr interface{}) error {
	return server.register(rcvr, name, true)
}

// the main registration routine
func (server *Server) register(rcvr interface{}, name string, useName bool) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !isExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	if _, present := server.serviceMap[sname]; present {
		return errors.New("rpc: service already defined: " + sname)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ, true)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}
	server.serviceMap[s.name] = s
	return nil
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			if reportErr {
				log.Println("method", mname, "has wrong number of ins:", mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Println(mname, "argument type not exported:", argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Println("method", mname, "reply type not a pointer:", replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Println("method", mname, "reply type not exported:", replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Println("method", mname, "has wrong number of outs:", mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Println("method", mname, "returns", returnType.String(), "not error")
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

// low level call to the actual method
func (s *service) call(server *Server, mtype *methodType, argv, replyv reflect.Value) error {

	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	if errInter != nil {
		return errInter.(error)
	}
	return nil
}

// execute a server side call
func (server *Server) Call(ServiceMethod string, in []interface{}, out interface{}) (err error) {

	// ensure that a pointer is passed
	rOut := reflect.ValueOf(out)
	if rOut.Kind() != reflect.Ptr || rOut.IsNil() {
		return errors.New("rpc: must be a non-nil pointer: " + ServiceMethod)
	}

	dot := strings.LastIndex(ServiceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + ServiceMethod)
		return
	}
	serviceName := ServiceMethod[:dot]
	methodName := ServiceMethod[dot+1:]

	// Look up the request.
	server.mu.RLock()
	service := server.serviceMap[serviceName]
	server.mu.RUnlock()
	if service == nil {
		err = errors.New("rpc: can't find service " + ServiceMethod)
		return
	}
	mtype := service.method[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + ServiceMethod)
		return
	}

	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	var argv reflect.Value
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}
	// argv guaranteed to be a pointer now.

	if argIsValue {
		argv = argv.Elem()
	}
	argt := argv.Type()

	// iterate through fields and set values
	n := argv.NumField()
	for i := 0; i < n; i += 1 {
		fieldValue := argv.Field(i)
		fieldType := argt.Field(i)
		argTag := fieldType.Tag.Get("arg")

		// erroneous tags
		if "" == argTag {
			panic(fmt.Sprintf("rpc: method: %q has empty arg:%q in its argument: %q", ServiceMethod, argTag, argt.Name()))
			//continue
		}
		j, err := strconv.Atoi(argTag)
		if nil != err {
			panic(fmt.Sprintf("rpc: method: %q has invalid arg:%q in its argument: %q", ServiceMethod, argTag, argt.Name()))
			//continue
		}
		if j < 0 {
			panic(fmt.Sprintf("rpc: method: %q has negative arg:%q in its argument: %q", ServiceMethod, argTag, argt.Name()))
			//continue
		}

		// if the value is outside the "in" slice it assumed to be an optional value
		if j >= n {
			// no error here, just skip
			continue
		}

		inType := reflect.TypeOf(in[j])
		if inType != fieldType.Type {
			panic(fmt.Sprintf("rpc: method: %q has type mismatch in[%d]:%v its struct needs:%v", ServiceMethod, j, inType, fieldType.Type))
			//continue
		}

		fieldValue.Set(reflect.ValueOf(in[j]))

	}

	// result type
	replyv := reflect.New(mtype.ReplyType.Elem())

	if err := service.call(server, mtype, argv, replyv); nil != err {
		return err
	}

	rOut.Elem().Set(replyv)
	return nil
}
