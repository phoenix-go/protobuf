package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"

	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

var registerStruct RegisterStruct

func GetRegister() *RegisterStruct {
	return &registerStruct
}

func init() {
	registerStruct.Init()
}

type RegisterStruct struct {
	registerNameByID map[string]string // key:msgID value:struct name
	registerIDByName map[string]string // key: struct name value:msgID
}

func (r *RegisterStruct) Init() {
	r.registerNameByID = make(map[string]string, 512)
	r.registerIDByName = make(map[string]string, 512)
}

func (r *RegisterStruct) Register(id, name string) {
	if n, ok := r.registerNameByID[id]; ok {
		fmt.Fprintf(os.Stderr, "Expect Register MsgID(%s) StructName(%s), But Already Used In StructName(%s)\n", id, name, n)
		return
	}

	r.registerNameByID[id] = name
	r.registerIDByName[name] = id
}

func (r *RegisterStruct) ResponseFile(pkgName string) *plugin.CodeGeneratorResponse_File {
	if len(r.registerNameByID) == 0 {
		return nil
	}
	fileName := "register.msg.go"
	content := r.autoGenerateRegister(pkgName)
	rf := &plugin.CodeGeneratorResponse_File{
		Name:    &fileName,
		Content: &content,
	}
	return rf
}

func (r *RegisterStruct) autoGenerateRegister(pkgName string) string {
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	fmt.Fprintf(buffer, "// Auto Generator Register. DO NOT EDIT.\n")
	fmt.Fprintf(buffer, "package %s\n\n", pkgName)
	fmt.Fprintf(buffer, "import (\n")
	fmt.Fprintf(buffer, "\"fmt\"\n")
	fmt.Fprintf(buffer, "\"github.com/gogo/protobuf/proto\"\n")
	fmt.Fprintf(buffer, ")\n\n")
	fmt.Fprintf(buffer, "var register = map[uint16]func() proto.Message {\n")
	for k, v := range r.registerNameByID {
		fmt.Fprintf(buffer, "uint16(%s): func() proto.Message { return &%s{} },\n", k, v)
	}
	fmt.Fprintf(buffer, "}\n\n")

	fmt.Fprintf(buffer, "func NewMessageByID(msgID uint16) (proto.Message, error) {\n")
	fmt.Fprintf(buffer, "if fn, ok := register[msgID]; ok {\n")
	fmt.Fprintf(buffer, "return fn(), nil\n}\n")
	fmt.Fprintf(buffer, "return nil, fmt.Errorf(\"Don't Contain MsgID(%%d) Data Structure.\", msgID)\n}\n")

	// fmt.Fprintf(os.Stderr, "Format Source:\n%s\n", buffer.String())

	data, err := format.Source(buffer.Bytes())
	if err != nil {
		panic(fmt.Sprintf("format register.msg.go error: %v", err))
	}

	return string(data)
}
