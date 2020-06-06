package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

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
	Package    *irgen.Package
	Includes   []*Include
	Elements   []Node // Function | GlobalVar | Comment | TypeDef
	MainFunc   *Function
	TypeDecls  []*TypeDecl
	TypeDefs   []*TypeDef
	StructDefs []*Struct
	UnionDefs  []*Union
	// To avoid duplicates, struct defs can be pre-announced.
	// This means they have not yet been defined, but they will be defined later on.
	StructDefsPre  map[string]bool
	tmpVarCount    int
	loopLabelStack []string
	loopCount      int
}

// Include ...
type Include struct {
	Path         string
	IsSystemPath bool
}

// Struct ...
type Struct struct {
	NodeBase
	Name   string
	Fields []*StructField
	Guard  string
}

// StructField ...
type StructField struct {
	Name string
	Type *TypeDecl
	// For fields of the kind `int arr[4]`
	Array string
}

// Union ...
type Union struct {
	NodeBase
	Name   string
	Fields []*StructField
	Guard  string
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
	IsExtern          bool
	Attributes        []string
}

// FunctionParameter ...
type FunctionParameter struct {
	Name string
	Type *TypeDecl
}

// FunctionType ...
type FunctionType struct {
	NodeBase
	Name       string
	ReturnType *TypeDecl
	Parameters []*FunctionParameter
}

// TypeDecl ...
type TypeDecl struct {
	NodeBase
	Code           string
	IsFunctionType bool
}

// TypeDef ...
type TypeDef struct {
	NodeBase
	Type string
	Name string
	// A guarded typedef is enclosed in #ifdef <Guard> #endif, because the
	// same type might be defined in multiple locations.
	Guard string
	// If true, `Type` includes the `Name`.
	IsFuncType bool
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

// GlobalVar ...
type GlobalVar struct {
	NodeBase
	Name string
	Type *TypeDecl
}

// Identifier ...
type Identifier struct {
	NodeBase
	Name string
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

// For ....
type For struct {
	NodeBase
	InitExpr      Node
	ConditionExpr Node
	LoopExpr      Node
	Body          []Node
}

// Break ....
type Break struct {
	NodeBase
}

// Continue ....
type Continue struct {
	NodeBase
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
	Type   *TypeDecl
	Values []Node
}

// UnionLiteral ...
type UnionLiteral struct {
	NodeBase
	Name  string
	Value Node
}

// Sizeof ...
type Sizeof struct {
	NodeBase
	Type *TypeDecl
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

// NewModule ...
func NewModule(p *irgen.Package) *Module {
	mod := &Module{StructDefsPre: make(map[string]bool), Package: p}
	mod.AddInclude("stdint.h", true)
	mod.AddInclude("stdbool.h", true)
	return mod
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

	str += "static uintptr_t g_zero = 0;\n\n"

	// Declarations of functions and global variables
	for _, n := range mod.Elements {
		if f, ok := n.(*Function); ok && (!f.IsExported || f.IsExtern) {
			str += f.Declaration("") + ";\n\n"
		} else if v, ok := n.(*GlobalVar); ok {
			str += v.Declaration("") + ";\n\n"
		}
	}

	// Function definitions
	for _, c := range mod.Elements {
		if f, ok := c.(*Function); ok && f.IsExtern {
			continue
		} else if _, ok := c.(*GlobalVar); ok {
			continue
		}
		str += c.ToString("") + ";\n\n"
	}

	// `main` function
	if mod.Package.TypePackage.IsExecutable() {
		str += "int main(int argc, char **argv) {\n"
		// Initialize all packages
		for _, p := range irgen.AllImports(mod.Package) {
			irf := p.Funcs[p.TypePackage.InitFunc]
			str += "    " + mangleFunctionName(p, irf.Name) + "();\n"
		}
		irf := mod.Package.Funcs[mod.Package.TypePackage.InitFunc]
		str += "    " + mangleFunctionName(mod.Package, irf.Name) + "();\n"
		// Call the Main function
		str += "    " + mod.MainFunc.Name + "();\n"
		str += "    return 0;\n}\n"
	}
	return str
}

// Header ...
func (mod *Module) Header(path string, filename string) string {
	mangledName := strings.ToUpper(mangleFileName(path, filename)) + "_H"
	str := ""
	str += "#ifndef " + mangledName + "\n"
	str += "#define " + mangledName + "\n\n"

	str += "#pragma clang diagnostic ignored \"-Wincompatible-library-redeclaration\"\n\n"
	// str += "#pragma GCC diagnostic ignored \"-Wincompatible-library-redeclaration\"\n\n"

	// Includes
	for _, inc := range mod.Includes {
		str += inc.ToString() + "\n"
	}
	str += "\n\n"

	// Declare types
	for _, t := range mod.TypeDecls {
		str += t.ToString("") + ";\n"
	}
	// Define types
	for _, t := range mod.TypeDefs {
		str += t.ToString("") + "\n"
	}
	// Define structs
	for _, s := range mod.StructDefs {
		str += s.ToString("") + ";\n"
	}
	// Define unions
	for _, s := range mod.UnionDefs {
		str += s.ToString("") + ";\n"
	}

	// Declarations of exported functions
	for _, n := range mod.Elements {
		if f, ok := n.(*Function); ok && f.IsExported {
			str += f.Declaration("") + ";\n"
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

// AddInclude ...
func (mod *Module) AddInclude(path string, isSystemPath bool) {
	if mod.HasInclude(path) {
		return
	}
	mod.Includes = append(mod.Includes, &Include{Path: path, IsSystemPath: isSystemPath})
}

func (mod *Module) startLoop() {
	mod.loopCount++
	n := "loop_" + strconv.Itoa(mod.loopCount)
	mod.loopLabelStack = append(mod.loopLabelStack, n)
}

func (mod *Module) loopLabel() string {
	return mod.loopLabelStack[len(mod.loopLabelStack)-1]
}

func (mod *Module) endLoop() {
	mod.loopLabelStack = mod.loopLabelStack[:len(mod.loopLabelStack)-1]
}

func (mod *Module) hasTypeDef(typename string) bool {
	for _, t := range mod.TypeDefs {
		if t.Name == typename {
			return true
		}
	}
	return false
}

func (mod *Module) addTypeDef(tdef *TypeDef) {
	mod.TypeDefs = append(mod.TypeDefs, tdef)
}

func (mod *Module) addTypeDecl(t *TypeDecl) {
	mod.TypeDecls = append(mod.TypeDecls, t)
}

func (mod *Module) addStructDefPre(name string) {
	mod.StructDefsPre[name] = true
}

func (mod *Module) addStructDef(s *Struct) {
	mod.StructDefs = append(mod.StructDefs, s)
}

func (mod *Module) hasStructDef(name string) bool {
	if _, ok := mod.StructDefsPre[name]; ok {
		return true
	}
	for _, s := range mod.StructDefs {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (mod *Module) addUnionDef(s *Union) {
	mod.UnionDefs = append(mod.UnionDefs, s)
}

func (mod *Module) addUnionDefPre(name string) {
	mod.StructDefsPre[name] = true
}

func (mod *Module) hasUnionDef(name string) bool {
	if _, ok := mod.StructDefsPre[name]; ok {
		return true
	}
	for _, s := range mod.UnionDefs {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (mod *Module) tmpVarName() string {
	mod.tmpVarCount++
	return "anon_" + strconv.Itoa(mod.tmpVarCount)
}

// ToString ...
func (n *Struct) ToString(indent string) string {
	str := ""
	if n.Guard != "" {
		str += indent + "#ifndef " + n.Guard + "\n"
		str += indent + "#define " + n.Guard + "\n"
	}
	str += indent + "struct " + n.Name + " {\n"
	for _, f := range n.Fields {
		str += indent + "    " + f.ToString() + ";\n"
	}
	str += indent + "}"
	if n.Guard != "" {
		str += "\n" + indent + "#endif\n"
	}
	return str
}

// ToString ...
func (n *StructField) ToString() string {
	return n.Type.ToString("") + " " + n.Name + n.Array
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

// TypeDef ...
func (n *FunctionType) TypeDef(typename string) *TypeDef {
	str := n.ReturnType.ToString("")
	str += " (*" + typename + ") ("
	for i, p := range n.Parameters {
		if i != 0 {
			str += ", "
		}
		str += p.ToString("")
	}
	str += ")"
	return &TypeDef{IsFuncType: true, Name: typename, Type: str}
}

// ToString ...
func (n *Union) ToString(indent string) string {
	str := ""
	if n.Guard != "" {
		str += indent + "#ifndef " + n.Guard + "\n"
		str += indent + "#define " + n.Guard + "\n"
	}
	str += indent + "union " + n.Name + " {\n"
	for _, f := range n.Fields {
		str += indent + "    " + f.ToString() + ";\n"
	}
	str += indent + "}"
	if n.Guard != "" {
		str += "\n" + indent + "#endif\n"
	}
	return str
}

// Declaration ...
func (n *Function) Declaration(indent string) string {
	str := ""
	if n.IsExtern {
		str = "extern "
	} else if !n.IsExported && !n.IsGenericInstance {
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
	attribs := ""
	for i, a := range n.Attributes {
		if i != 0 {
			attribs += ", " + a
		} else {
			attribs += a
		}
	}
	if attribs != "" {
		str += " __attribute__((" + attribs + "))"
	}
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

// NewTypeDef ...
func NewTypeDef(typ string, name string) *TypeDef {
	return &TypeDef{Type: typ, Name: name}
}

// ToString ...
func (n *TypeDef) ToString(indent string) string {
	str := ""
	if n.Guard != "" {
		str += indent + "#ifndef " + n.Guard + "\n"
		str += indent + "#define " + n.Guard + "\n"
	}
	if n.IsFuncType {
		str += indent + "typedef " + n.Type + ";"
	} else {
		str += indent + "typedef " + n.Type + " " + n.Name + ";"
	}
	if n.Guard != "" {
		str += "\n" + indent + "#endif\n"
	}
	return str
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
	return indent + "return"
}

// ToString ...
func (n *Unary) ToString(indent string) string {
	if n.Operator == "sizeof" {
		return indent + "sizeof(" + n.Expr.ToString("") + ")"
	}
	if n.Precedence() <= n.Expr.Precedence() {
		if n.Operator == "++" {
			return indent + n.Expr.ToString("") + "++"
		}
		return indent + n.Operator + n.Expr.ToString("")
	}
	if n.Operator == "++" {
		return indent + n.Expr.ToString("") + "++"
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
	case "[", ".", "->":
		return 1
	case "*", "/", "%":
		return 3
	case "-", "+":
		return 4
	case "<<", ">>":
		return 5
	case "<", ">", "<=", ">=":
		return 6
	case "==", "!=":
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
	case "=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|=":
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
	for i, arg := range n.Args {
		if i > 0 {
			str += ", "
		}
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

// Declaration ...
func (n *GlobalVar) Declaration(indent string) string {
	str := indent
	if !isUpperCaseName(n.Name) {
		str += "static "
	}
	str += n.Type.ToString("") + " " + n.Name
	//	if n.ConstExpr != nil {
	//		str += " = " + n.ConstExpr.ToString("")
	//	}
	return str
}

// ToString ...
func (n *GlobalVar) ToString(indent string) string {
	return ""
}

// ToString ...
func (n *Identifier) ToString(indent string) string {
	return indent + n.Name
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
		str += b.ToString(indent+"    ") + ";\n"
	}
	str += indent + "}"
	return str
}

// ToString ...
func (n *For) ToString(indent string) string {
	str := indent + "for ("
	if n.InitExpr != nil {
		str += n.InitExpr.ToString("") + "; "
	} else {
		str += ";"
	}
	if n.ConditionExpr != nil {
		str += n.ConditionExpr.ToString("") + "; "
	} else {
		str += ";"
	}
	if n.LoopExpr != nil {
		str += n.LoopExpr.ToString("")
	}
	str += ") {\n"
	for _, b := range n.Body {
		str += b.ToString(indent+"    ") + ";\n"
	}
	str += indent + "}"
	return str
}

// ToString ...
func (n *Break) ToString(indent string) string {
	return indent + "break"
}

// ToString ...
func (n *Continue) ToString(indent string) string {
	return indent + "continue"
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
	str := indent + "(" + n.Type.ToString("") + "){"
	for i, v := range n.Values {
		if i != 0 {
			str += ", "
		}
		str += v.ToString("")
	}
	// Empty initializer lists are not allowed in C
	if len(n.Values) == 0 {
		str += "0"
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

// Precedence ...
func (n *Sizeof) Precedence() int {
	return 1
}

// ToString ...
func (n *Sizeof) ToString(indent string) string {
	return indent + "sizeof(" + n.Type.ToString("") + ")"
}

func isUpperCaseName(name string) bool {
	for _, r := range name {
		if unicode.ToUpper(r) == r {
			return true
		}
		return false
	}
	return false
}
