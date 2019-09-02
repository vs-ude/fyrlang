package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Node ...
type Node interface {
	ToString(indent string) string
	Precedence() int
}

// NodeBase ...
type NodeBase struct {
}

// Module ...
type Module struct {
	Package  *irgen.Package
	Includes []*Include
	Strings  map[string]*String
	Elements []Node // Struct | Function | Var | Comment | TypeDecl | Extern
}

// Include ...
type Include struct {
	Path         string
	IsSystemPath bool
}

// Extern ...
type Extern struct {
	NodeBase
	Var *Var
}

// String ...
type String struct {
	NodeBase
	Value      string
	Identifier string
}

// Struct ...
type Struct struct {
	NodeBase
	Name   string
	Fields []*StructField
}

// StructField ...
type StructField struct {
	Name string
	Type *TypeDecl
}

// Function ...
type Function struct {
	NodeBase
	Name              string
	ReturnType        *TypeDecl
	Parameters        []*FunctionParameter
	Body              []Node
	IsGenericInstance bool
	IsExported        bool
}

// FunctionParameter ...
type FunctionParameter struct {
	Name string
	Type *TypeDecl
}

// TypeDecl ...
type TypeDecl struct {
	NodeBase
	Code           string
	IsFunctionType bool
}

// Return ...
type Return struct {
	NodeBase
	Expr Node
}

// Unary ...
type Unary struct {
	NodeBase
	Expr     Node
	Operator string
}

// Binary ...
type Binary struct {
	NodeBase
	Left     Node
	Right    Node
	Operator string
}

// FunctionCall ...
type FunctionCall struct {
	NodeBase
	FuncExpr Node
	Args     []Node
}

// TypeCast ...
type TypeCast struct {
	NodeBase
	Type *TypeDecl
	Expr Node
}

// Var ...
type Var struct {
	NodeBase
	Name     string
	Type     *TypeDecl
	InitExpr Node
}

// Constant ...
type Constant struct {
	NodeBase
	Code string
}

// Comment ...
type Comment struct {
	NodeBase
	Text string
}

// If ....
type If struct {
	NodeBase
	Expr       Node
	Body       []Node
	ElseClause *Else
}

// Else ...
type Else struct {
	NodeBase
	Body []Node
}

// Label ...
type Label struct {
	NodeBase
	Name string
}

// Goto ...
type Goto struct {
	NodeBase
	Name string
}

// CompoundLiteral ...
type CompoundLiteral struct {
	NodeBase
	Values []Node
}

// UnionLiteral ...
type UnionLiteral struct {
	NodeBase
	Name  string
	Value Node
}

func mangleFileName(path string, filename string) string {
	sum := sha256.Sum256([]byte(path))
	sumHex := hex.EncodeToString(sum[:])
	return filename + "_" + sumHex
}

// Precedence ...
func (n *NodeBase) Precedence() int {
	return 0
}

// ToString ...
func (n *Include) ToString() string {
	if n.IsSystemPath {
		return "#include <" + n.Path + ">"
	}
	return "#include \"" + n.Path + "\""
}

// Implementation ...
func (mod *Module) Implementation(path string, filename string) string {
	headerFile := ""
	if mod.Package.TypePackage.IsInFyrPath() {
		headerFile = filepath.Join(pkgOutputPath(mod.Package), filename)
	} else {
		headerFile = filepath.Join(pkgOutputPath(mod.Package), filename)
	}
	str := ""
	str += "#include \"" + headerFile + ".h\"\n"
	str += "\n"

	for _, s := range mod.Strings {
		str += s.ToString("") + "\n\n"
	}
	for _, n := range mod.Elements {
		if f, ok := n.(*Function); ok && !f.IsExported {
			str += f.Declaration("") + ";\n\n"
		}
	}

	for _, c := range mod.Elements {
		if _, ok := c.(*TypeDecl); ok {
			// Do nothing
		} else if f, ok := c.(*Function); ok {
			str += f.ToString("") + "\n\n"
		} else if e, ok := c.(*Extern); ok {
			str += e.Var.ToString("") + ";\n\n"
		} else {
			str += c.ToString("") + ";\n\n"
		}
	}

	if mod.Package.TypePackage.IsExecutable() {
		str += "int main(int argc, char **argv) {\n    return 0;\n}\n"
	}
	return str
}

// Header ...
func (mod *Module) Header(path string, filename string) string {
	mangledName := strings.ToUpper(mangleFileName(path, filename)) + "_H"
	str := ""
	str += "#ifndef " + mangledName + "\n"
	str += "#define " + mangledName + "\n\n"

	for _, inc := range mod.Includes {
		str += inc.ToString() + "\n"
	}
	str += "\n\n"

	for _, n := range mod.Elements {
		if t, ok := n.(*TypeDecl); ok {
			str += t.ToString("") + ";\n"
		}
	}
	for _, n := range mod.Elements {
		if f, ok := n.(*Function); ok && f.IsExported {
			str += f.Declaration("") + ";\n"
		}
	}

	// Export global variables
	for _, n := range mod.Elements {
		if e, ok := n.(*Extern); ok {
			str += e.ToString("") + ";\n"
		}
	}

	str += "\n#endif\n"
	return str
}

// HasInclude ...
func (mod *Module) HasInclude(path string) bool {
	for _, inc := range mod.Includes {
		if inc.Path == path {
			return true
		}
	}
	return false
}

// AddString ...
func (mod *Module) AddString(str string) *String {
	if s, ok := mod.Strings[str]; ok {
		return s
	}
	sum := sha256.Sum256([]byte(str))
	mangledValue := hex.EncodeToString(sum[:])
	ident := "str_" + mangledValue
	s := &String{Value: str, Identifier: ident}
	mod.Strings[str] = s
	return s
}

// NewExtern ...
func NewExtern(v *Var) *Extern {
	return &Extern{Var: v}
}

// ToString ...
func (n *Extern) ToString(indent string) string {
	return indent + "extern " + n.Var.ToString("")
}

// ToString ...
func (n *String) ToString(indent string) string {
	str := indent + "struct {\n" + indent + "    int_t size;\n" + indent + "    int_t lockcount;\n" + indent + "    int_t refcount;\n" + indent + "    uint8_t data[" + strconv.Itoa(len(n.Value)) + "];\n" + indent + "} " + n.Identifier + " __attribute((weak))__ = {" + strconv.Itoa(len(n.Value)) + ", 1, 1"
	if n.Value != "" {
		str += ","
	}
	for i := 0; i < len(n.Value); i++ {
		if i != 0 {
			str += ","
		}
		str += strconv.Itoa(int(n.Value[i]))
	}
	return str + "};"
}

// ToString ...
func (n *Struct) ToString(indent string) string {
	str := indent + "struct " + n.Name + " {\n"
	for _, f := range n.Fields {
		str += indent + "    " + f.ToString() + "\n"
	}
	str += indent + "}"
	return str
}

// ToString ...
func (n *StructField) ToString() string {
	return n.Type.ToString("") + " " + n.Name
}

// ToString ...
func (n *Function) ToString(indent string) string {
	str := ""
	if !n.IsExported && !n.IsGenericInstance {
		str = "static "
	}
	str += indent + n.ReturnType.ToString("")
	if n.IsGenericInstance {
		str += " __attribute__((weak))"
	}
	str += " " + n.Name + "("
	for i, p := range n.Parameters {
		if i != 0 {
			str += ", "
		}
		str += p.ToString("")
	}
	str += ") {\n"
	for _, b := range n.Body {
		str += b.ToString(indent+"    ") + ";\n"
	}
	return str + "\n" + indent + "}"
}

// Declaration ...
func (n *Function) Declaration(indent string) string {
	str := ""
	if !n.IsExported && !n.IsGenericInstance {
		str = "static "
	}
	str += indent + n.ReturnType.ToString("")
	if n.IsGenericInstance {
		str += " __attribute__((weak))"
	}
	str += " " + n.Name + "("
	for i, p := range n.Parameters {
		if i != 0 {
			str += ", "
		}
		str += p.ToString("")
	}
	str += ")"
	return str
}

// ToString ...
func (n *FunctionParameter) ToString(indent string) string {
	if n.Type.IsFunctionType {
		str := n.Type.ToString("")
		i := strings.Index(str, "(*)")
		return str[:i] + "(*" + n.Name + ")" + str[i+3:]
	}
	return n.Type.ToString("") + " " + n.Name
}

// NewTypeDecl ...
func NewTypeDecl(code string) *TypeDecl {
	return &TypeDecl{Code: code}
}

// ToString ...
func (n *TypeDecl) ToString(indent string) string {
	return indent + n.Code
}

// NewFunctionType ...
func NewFunctionType(returnType *TypeDecl, parameters []*TypeDecl) *TypeDecl {
	str := ""
	if returnType != nil {
		str += returnType.Code
	} else {
		str += "void"
	}
	str += "(*)("
	for i, p := range parameters {
		if i != 0 {
			str += ", "
		}
		str += p.Code
	}
	str += ")"
	return &TypeDecl{Code: str, IsFunctionType: true}
}

// ToString ...
func (n *Return) ToString(indent string) string {
	if n.Expr != nil {
		return indent + "return " + n.Expr.ToString("")
	}
	return "return"
}

// ToString ...
func (n *Unary) ToString(indent string) string {
	if n.Operator == "sizeof" {
		return indent + "sizeof(" + n.Expr.ToString("") + ")"
	}
	if n.Precedence() <= n.Expr.Precedence() {
		return indent + n.Operator + n.Expr.ToString("")
	}
	return indent + n.Operator + n.Expr.ToString("")
}

// Precedence ...
func (n *Unary) Precedence() int {
	return 2
}

// ToString ...
func (n *Binary) ToString(indent string) string {
	str := indent
	if n.Precedence() <= n.Left.Precedence() {
		str += "(" + n.Left.ToString("") + ")"
	} else {
		str += n.Left.ToString("")
	}
	if n.Operator == "." {
		str += "."
	} else if n.Operator == "[" {
		str += "["
	} else {
		str += " " + n.Operator + " "
	}
	if n.Precedence() <= n.Right.Precedence() {
		str += "(" + n.Right.ToString("") + ")"
	} else {
		str += n.Right.ToString("")
	}
	if n.Operator == "[" {
		str += "]"
	}
	return str
}

// Precedence ...
func (n *Binary) Precedence() int {
	switch n.Operator {
	case "[":
	case ".":
	case "->":
		return 1
	case "*":
	case "/":
	case "%":
		return 3
	case "-":
	case "+":
		return 4
	case "<<":
	case ">>":
		return 5
	case "<":
	case ">":
	case "<=":
	case ">=":
		return 6
	case "==":
	case "!=":
		return 7
	case "&":
		return 8
	case "^":
		return 9
	case "|":
		return 10
	case "&&":
		return 11
	case "||":
		return 12
	case "=":
	case "+=":
	case "-=":
	case "*=":
	case "/=":
	case "%=":
	case "<<=":
	case ">>=":
	case "&=":
	case "^=":
	case "|=":
		return 13
	}
	panic("Ooooops")
}

// ToString ...
func (n *FunctionCall) ToString(indent string) string {
	str := indent
	if n.Precedence() <= n.FuncExpr.Precedence() {
		str += "(" + n.FuncExpr.ToString("") + ")"
	} else {
		str += n.FuncExpr.ToString("")
	}
	str += "("
	for _, arg := range n.Args {
		str += arg.ToString("")
	}
	str += ")"
	return str
}

// Precedence ...
func (n *FunctionCall) Precedence() int {
	return 1
}

// ToString ...
func (n *TypeCast) ToString(indent string) string {
	if n.Precedence() <= n.Expr.Precedence() {
		return indent + "(" + n.Type.ToString("") + ")(" + n.Expr.ToString("") + ")"
	}
	return indent + "(" + n.Type.ToString("") + ")" + n.Expr.ToString("")
}

// Precedence ...
func (n *TypeCast) Precedence() int {
	return 2
}

// ToString ...
func (n *Var) ToString(indent string) string {
	str := indent + n.Type.ToString("") + " " + n.Name
	if n.InitExpr != nil {
		str += " = " + n.InitExpr.ToString("")
	}
	return str
}

// ToString ...
func (n *Constant) ToString(indent string) string {
	return indent + n.Code
}

// ToString ...
func (n *Comment) ToString(indent string) string {
	return indent + "/*" + n.Text + "*/"
}

// ToString ...
func (n *If) ToString(indent string) string {
	str := indent + "if (" + n.Expr.ToString("") + ") {\n"
	for _, b := range n.Body {
		str += b.ToString(indent+"    ") + ";\n"
	}
	str += indent + "}"
	if n.ElseClause != nil {
		str += " " + n.ElseClause.ToString(indent)
	}
	return str
}

// ToString ...
func (n *Else) ToString(indent string) string {
	str := "else {\n"
	for _, b := range n.Body {
		str += b.ToString("") + ";\n"
	}
	str += indent + "}"
	return str
}

// ToString ...
func (n *Label) ToString(indent string) string {
	return indent + n.Name + ":"
}

// ToString ...
func (n *Goto) ToString(indent string) string {
	return indent + "goto " + n.Name
}

// ToString ...
func (n *CompoundLiteral) ToString(indent string) string {
	str := indent + "{"
	for i, v := range n.Values {
		if i != 0 {
			str += ", "
		}
		str += v.ToString("")
	}
	str += "}"
	return str
}

// Precedence ...
func (n *CompoundLiteral) Precedence() int {
	return 1
}

// ToString ...
func (n *UnionLiteral) ToString(indent string) string {
	return indent + "{ ." + n.Name + " = " + n.Value.ToString("") + "}"
}

// Precedence ...
func (n *UnionLiteral) Precedence() int {
	return 1
}
