package generator

import (
	"fmt"
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
	r.registerNameByID[name] = id
}

func (r *RegisterStruct) ResponseFile() *plugin.CodeGeneratorResponse_File {
	rf := &plugin.CodeGeneratorResponse_File{}

	return rf
}
