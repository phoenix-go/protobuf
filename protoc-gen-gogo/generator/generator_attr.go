package generator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// invoked after method generateMessage in generator/generator.go
func (g *Generator) generateStructCommentCommand(mc *msgCtx) bool {
	cmds := g.filterCommentCmd(mc)
	if len(cmds) == 0 {
		return false
	}

	for _, cmdStr := range cmds {
		g.parseCmdLine(mc, cmdStr)
	}
	return true
}

func (g *Generator) filterCommentCmd(mc *msgCtx) []string {
	commentStr := g.Comments(mc.message.path)
	commentsSlc := strings.Split(commentStr, "\n")
	commentCmd := make([]string, 0, len(commentsSlc))
	for _, str := range commentsSlc {
		if strings.HasPrefix(str, "go:attr") {
			commentCmd = append(commentCmd, str)
		}
	}
	return commentCmd
}

func (g *Generator) parseCmdLine(mc *msgCtx, cmdStr string) {
	var (
		word string
		lx   *lexer
	)
	lx = makeLexer(cmdStr)
	if word = lx.word(); word != "go:attr" {
		panic(fmt.Sprintf("Comment Command Error [%d,%d] (%s) in (%s)\n", lx.start, lx.pos, word, cmdStr))
	}
	lx.truncate()
	lx.omitSpace()
	for !lx.eof() {
		word = lx.wordByCond('=')
		lx.omitByCond('=')
		switch word {
		case "method":
			g.generatorMethod(mc, lx)
		default:
			panic(fmt.Sprintf("Commnet Command Invalid parameter[%s] in [%s]\n", word, cmdStr))
		}
	}
}

type typeValue struct {
	t string
	v string
}

type methodDescribe struct {
	name string
	ins  []typeValue
	outs []typeValue
}

func (g *Generator) generatorMethod(mc *msgCtx, lx *lexer) {
	var (
		r    rune
		word string
	)
	if r = lx.Rune(); r != rune('"') {
		panic("Method paramter format error, not a string")
	}

	lx.truncate()
	lx.omitSpace()
	desc := &methodDescribe{}
	for !lx.eof() {
		r = lx.Rune()
		if r == rune('"') {
			break
		}
		lx.cancel()
		word = lx.wordByCond(':')
		lx.omitByCond(':')
		switch word {
		case "name":
			desc.name = lx.wordByCond(',')
		case "in":
			desc.ins = g.generateMethodInOut(lx)
		case "out":
			desc.outs = g.generateMethodInOut(lx)
		default:
			panic(fmt.Sprintf("method attr error (%s)", word))
		}
		lx.omitByCond(',')
	}

	lx.omitByCond('"')

	// 打印对应的函数处理方法
	g.printCommentCmdMethod(mc, desc)
}

func (g *Generator) printCommentCmdMethod(mc *msgCtx, desc *methodDescribe) {
	g.p("func(m *", mc.goName, ") ", desc.name, "(")
	for i, in := range desc.ins {
		g.p(in.v, " ", in.t)
		if i != len(desc.ins)-1 {
			g.p(", ")
		}
	}
	g.p(")")
	for i, out := range desc.outs {
		if i == 0 && len(desc.outs) != 1 {
			g.p("(")
		}

		g.p(out.t)

		if len(desc.outs) != 1 {
			if i == len(desc.outs)-1 {
				g.p(")")
			} else {
				g.p(", ")
			}
		}
	}

	g.P("{")

	g.p("return ")

	for i, out := range desc.outs {
		g.p(out.v)
		if i != len(desc.outs)-1 {
			g.p(", ")
		}
	}

	g.P()
	g.P("}")
	g.P()
}

// P prints the arguments to the generated output.  It handles strings and int32s, plus
// handling indirections because they may be *string, etc.  Any inputs of type AnnotatedAtoms may emit
// annotations in a .meta file in addition to outputting the atoms themselves (if g.annotateCode
// is true).
func (g *Generator) p(str ...interface{}) {
	if !g.writeOutput {
		return
	}
	g.WriteString(g.indent)
	for _, v := range str {
		switch v := v.(type) {
		case *AnnotatedAtoms:
			begin := int32(g.Len())
			for _, v := range v.atoms {
				g.printAtom(v)
			}
			if g.annotateCode {
				end := int32(g.Len())
				var path []int32
				for _, token := range strings.Split(v.path, ",") {
					val, err := strconv.ParseInt(token, 10, 32)
					if err != nil {
						g.Fail("could not parse proto AST path: ", err.Error())
					}
					path = append(path, int32(val))
				}
				g.annotations = append(g.annotations, &descriptor.GeneratedCodeInfo_Annotation{
					Path:       path,
					SourceFile: &v.source,
					Begin:      &begin,
					End:        &end,
				})
			}
		default:
			g.printAtom(v)
		}
	}
}

func (g *Generator) generateMethodInOut(lx *lexer) []typeValue {
	var r rune
	if r = lx.Rune(); r != rune('[') {
		panic(fmt.Sprintf("MethodInOut error, invalid format"))
	}
	lx.omitByCond('[')
	for !lx.eof() {
		if r = lx.Rune(); r == rune(']') {
			lx.cancel()
			break
		}
	}
	paramStr := lx.data[lx.start:lx.pos]
	paramsSlc := strings.Split(paramStr, ",")
	typeValeSlc := make([]typeValue, 0, len(paramsSlc))
	for _, param := range paramsSlc {
		tvSlc := strings.Split(param, "=")
		if len(tvSlc) != 2 {
			continue
		}
		typeValeSlc = append(typeValeSlc, typeValue{tvSlc[0], tvSlc[1]})
	}
	lx.omitByCond(']')

	return typeValeSlc
}

type lexer struct {
	data   string
	start  int
	pos    int
	offset int
	// stack
}

func makeLexer(d string) *lexer {
	lx := &lexer{
		data:  d,
		start: 0,
		pos:   0,
	}
	return lx
}

func (lx *lexer) eof() bool {
	return lx.start >= len(lx.data)
}

func (lx *lexer) Rune() rune {
	var r rune
	r, lx.offset = utf8.DecodeRuneInString(lx.data[lx.pos:])
	lx.pos += lx.offset
	return r
}

func (lx *lexer) cancel() {
	lx.pos -= lx.offset
	lx.offset = 0
}

func (lx *lexer) truncate() {
	lx.start = lx.pos
}

func (lx *lexer) omitByCond(rs ...rune) {
	check := func(r rune) bool {
		for _, cr := range rs {
			if r == cr {
				return true
			}
		}
		return false
	}

	for lx.pos < len(lx.data) {
		if r := lx.Rune(); !unicode.IsSpace(r) && !check(r) {
			lx.cancel()
			break
		}
	}
	lx.truncate()
}

func (lx *lexer) omitSpace() {
	lx.omitByCond()
}

func (lx *lexer) word() string {
	return lx.wordByCond()
}

func (lx *lexer) wordByCond(rs ...rune) string {

	check := func(r rune) bool {
		for _, cr := range rs {
			if r == cr {
				return true
			}
		}
		return false
	}

	for lx.pos < len(lx.data) {
		r := lx.Rune()
		if unicode.IsSpace(r) || check(r) {
			lx.cancel()
			break
		}
	}
	return lx.data[lx.start:lx.pos]
}
