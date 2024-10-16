package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// 包含了一个方法的完整信息
type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 第一个参数的类型
	ReplyType reflect.Type   // 第二个参数的类型
	numCalls  uint64         // 后续统计方法调用次数时会用到
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 创建对应类型实例
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// arg may be a pointer type, or a value type
	// only ptr can use Elem()
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string                 // 映射的结构体的名称
	typ    reflect.Type           // 结构体的类型
	rcvr   reflect.Value          // 结构体的实例本身
	method map[string]*methodType // 存储映射的结构体的所有符合条件的方法
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name() // 如果不使用Indirect，rcvr传指针会输出*MyService，但是需要是MyService，通过Indirect可以解引
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) { // 检查服务名称是否以大写字母开头
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 { // 第一个参数类似python的self
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() { // 检查返回值类型是否为 error
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	//fmt.Println(ast.IsExported(t.Name()), t.PkgPath() == "")
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// 通过反射调用函数
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv}) // {方法实例，参数1，参数2}
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
