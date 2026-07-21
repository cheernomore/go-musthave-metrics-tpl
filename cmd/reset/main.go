// Command reset генерирует методы Reset() для структур, помеченных
// комментарием // generate:reset.
//
// # Назначение
//
// Утилита сканирует все пакеты, начиная с указанной (по умолчанию текущей)
// директории и ниже, находит структуры с комментарием // generate:reset
// и для каждой генерирует метод Reset(), сбрасывающий поля структуры к
// начальным значениям. Сгенерированные методы для структур одного пакета
// помещаются в файл reset.gen.go этого пакета.
//
// # Правила сброса
//
//   - примитивы приводятся к нулевым значениям (0, "", false);
//   - слайсы обрезаются по длине: s = s[:0];
//   - мапы очищаются встроенным clear;
//   - вложенные структуры с методом Reset() — вызывают его;
//   - не nil указатели сбрасывают значение по тем же правилам.
//
// # Запуск
//
//	go run ./cmd/reset          # сканирует текущую директорию и ниже
//	go run ./cmd/reset ./path   # сканирует указанную директорию
//
// Для поиска структур используется пакет go/ast.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/imports"
)

// marker — комментарий-маркер, помечающий структуры для генерации Reset().
const marker = "generate:reset"

// generatedFile — имя файла со сгенерированными методами.
const generatedFile = "reset.gen.go"

// dirsToSkip — каталоги, которые не сканируются (кроме корневого).
var dirsToSkip = map[string]bool{
	"vendor":   true,
	"testdata": true,
	".git":     true,
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := generate(root); err != nil {
		log.Fatalf("reset: %v", err)
	}
}

// generate сканирует дерево каталогов начиная с root и для каждого пакета
// с помеченными структурами создаёт файл reset.gen.go.
func generate(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && (dirsToSkip[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		return processDir(path)
	})
}

// processDir разбирает Go-файлы каталога dir, группирует их по пакетам и при
// наличии помеченных структур записывает для пакета reset.gen.go.
func processDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	// os.ReadDir возвращает записи в отсортированном порядке, поэтому файлы
	// внутри пакета собираются детерминированно.
	filesByPkg := make(map[string][]*ast.File)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if name == generatedFile || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("разбор %s: %w", name, err)
		}
		filesByPkg[f.Name.Name] = append(filesByPkg[f.Name.Name], f)
	}

	for _, pkgName := range sortedKeys(filesByPkg) {
		g := newGenerator(fset, filesByPkg[pkgName])
		if len(g.targets) == 0 {
			continue
		}
		if err := g.write(dir, pkgName); err != nil {
			return err
		}
	}
	return nil
}

// target — структура, для которой нужно сгенерировать метод Reset().
type target struct {
	name string
	st   *ast.StructType
}

// generator хранит состояние генерации для одного пакета.
type generator struct {
	fset *token.FileSet
	// resettable — имена типов пакета, у которых есть (или будет) метод Reset().
	resettable map[string]bool
	// typeDefs — имена типов пакета и их базовые (underlying) выражения.
	typeDefs map[string]ast.Expr
	// targets — структуры с маркером в порядке обхода файлов.
	targets []target
}

func newGenerator(fset *token.FileSet, files []*ast.File) *generator {
	g := &generator{
		fset:       fset,
		resettable: make(map[string]bool),
		typeDefs:   make(map[string]ast.Expr),
	}

	// Первый проход: собираем определения типов, существующие методы Reset()
	// и помеченные структуры.
	for _, f := range files {
		g.collect(f)
	}
	return g
}

// collect наполняет состояние генератора по одному файлу.
func (g *generator) collect(f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isResetMethod(d) {
				g.resettable[receiverType(d)] = true
			}
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			declMarked := hasMarker(d.Doc)
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				g.typeDefs[ts.Name.Name] = ts.Type
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if declMarked || hasMarker(ts.Doc) {
					g.resettable[ts.Name.Name] = true
					g.targets = append(g.targets, target{name: ts.Name.Name, st: st})
				}
			}
		}
	}
}

// write формирует и записывает reset.gen.go в каталоге dir.
func (g *generator) write(dir, pkgName string) error {
	var b strings.Builder
	b.WriteString("// Code generated by \"go run ./cmd/reset\"; DO NOT EDIT.\n\n")
	b.WriteString("package " + pkgName + "\n\n")

	for _, t := range g.targets {
		b.WriteString(g.method(t))
	}

	outPath := filepath.Join(dir, generatedFile)
	formatted, err := imports.Process(outPath, []byte(b.String()), nil)
	if err != nil {
		return fmt.Errorf("форматирование %s: %w", outPath, err)
	}
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return fmt.Errorf("запись %s: %w", outPath, err)
	}
	return nil
}

// method генерирует исходный код метода Reset() для одной структуры.
func (g *generator) method(t target) string {
	recv := receiverName(t.name)

	var stmts []string
	for _, field := range t.st.Fields.List {
		for _, name := range fieldNames(field) {
			stmts = append(stmts, g.reset(recv+"."+name, field.Type)...)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "func (%s *%s) Reset() {\n", recv, t.name)
	fmt.Fprintf(&b, "\tif %s == nil {\n\t\treturn\n\t}\n", recv)
	if len(stmts) > 0 {
		b.WriteString("\n")
		for _, s := range stmts {
			b.WriteString("\t" + s + "\n")
		}
	}
	b.WriteString("}\n\n")
	return b.String()
}

// reset возвращает строки кода, сбрасывающие поле expr типа typ.
func (g *generator) reset(expr string, typ ast.Expr) []string {
	switch t := typ.(type) {
	case *ast.Ident:
		return g.resetNamed(expr, t.Name, false)
	case *ast.StarExpr:
		inner := g.resetPointee(expr, t.X)
		if len(inner) == 0 {
			return nil
		}
		return wrapNotNil(expr, inner)
	case *ast.ArrayType:
		if t.Len == nil { // слайс
			return []string{expr + " = " + expr + "[:0]"}
		}
		return []string{expr + " = " + g.typeString(typ) + "{}"} // массив
	case *ast.MapType:
		return []string{"clear(" + expr + ")"}
	case *ast.InterfaceType, *ast.ChanType, *ast.FuncType:
		return []string{expr + " = nil"}
	case *ast.SelectorExpr:
		return []string{expr + " = " + g.typeString(typ) + "{}"}
	default:
		return nil
	}
}

// resetPointee возвращает строки, сбрасывающие значение по указателю expr
// (внутри блока, где expr уже проверен на nil). elem — тип, на который
// указывает expr.
func (g *generator) resetPointee(expr string, elem ast.Expr) []string {
	switch e := elem.(type) {
	case *ast.Ident:
		return g.resetNamed(expr, e.Name, true)
	case *ast.ArrayType:
		if e.Len == nil {
			return []string{"*" + expr + " = (*" + expr + ")[:0]"}
		}
		return []string{"*" + expr + " = " + g.typeString(elem) + "{}"}
	case *ast.MapType:
		return []string{"clear(*" + expr + ")"}
	case *ast.StarExpr:
		inner := g.resetPointee("(*"+expr+")", e.X)
		if len(inner) == 0 {
			return nil
		}
		return wrapNotNil("*"+expr, inner)
	case *ast.InterfaceType, *ast.ChanType, *ast.FuncType:
		return []string{"*" + expr + " = nil"}
	case *ast.SelectorExpr:
		return []string{"*" + expr + " = " + g.typeString(elem) + "{}"}
	default:
		return nil
	}
}

// resetNamed обрабатывает поле, тип которого задан идентификатором name.
// ptr указывает, что expr — указатель на значение этого типа.
func (g *generator) resetNamed(expr, name string, ptr bool) []string {
	switch {
	case isBasicType(name):
		if ptr {
			return []string{"*" + expr + " = " + zeroValue(name)}
		}
		return []string{expr + " = " + zeroValue(name)}
	case name == "any" || name == "error":
		if ptr {
			return []string{"*" + expr + " = nil"}
		}
		return []string{expr + " = nil"}
	case g.resettable[name]:
		// Тип имеет метод Reset(): вызываем его (указатель — на сам expr).
		return []string{expr + ".Reset()"}
	default:
		// Тип, определённый в этом пакете, без Reset(): сбрасываем по его
		// базовому типу, если он известен, иначе обнуляем литералом.
		if under, ok := g.typeDefs[name]; ok {
			if stmts := g.resetUnderlying(expr, name, under, ptr); stmts != nil {
				return stmts
			}
		}
		if ptr {
			return []string{"*" + expr + " = " + name + "{}"}
		}
		return []string{expr + " = " + name + "{}"}
	}
}

// resetUnderlying сбрасывает поле именованного типа name по его базовому типу.
func (g *generator) resetUnderlying(expr, name string, under ast.Expr, ptr bool) []string {
	switch u := under.(type) {
	case *ast.Ident:
		if isBasicType(u.Name) {
			if ptr {
				return []string{"*" + expr + " = " + zeroValue(u.Name)}
			}
			return []string{expr + " = " + zeroValue(u.Name)}
		}
	case *ast.ArrayType:
		// Именованный слайс обрезается по длине, именованный массив обнуляется.
		if u.Len == nil {
			if ptr {
				return []string{"*" + expr + " = (*" + expr + ")[:0]"}
			}
			return []string{expr + " = " + expr + "[:0]"}
		}
		if ptr {
			return []string{"*" + expr + " = " + name + "{}"}
		}
		return []string{expr + " = " + name + "{}"}
	case *ast.StructType:
		if ptr {
			return []string{"*" + expr + " = " + name + "{}"}
		}
		return []string{expr + " = " + name + "{}"}
	case *ast.MapType:
		if ptr {
			return []string{"clear(*" + expr + ")"}
		}
		return []string{"clear(" + expr + ")"}
	}
	return nil
}

// typeString печатает выражение типа обратно в исходный код.
func (g *generator) typeString(e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, g.fset, e)
	return b.String()
}

// wrapNotNil оборачивает строки проверкой "if expr != nil { ... }".
func wrapNotNil(expr string, inner []string) []string {
	out := make([]string, 0, len(inner)+2)
	out = append(out, "if "+expr+" != nil {")
	for _, s := range inner {
		out = append(out, "\t"+s)
	}
	return append(out, "}")
}

// fieldNames возвращает имена доступа к полю: имена обычных полей либо имя
// встроенного (embedded) поля.
func fieldNames(field *ast.Field) []string {
	if len(field.Names) > 0 {
		names := make([]string, 0, len(field.Names))
		for _, n := range field.Names {
			names = append(names, n.Name)
		}
		return names
	}
	if n := embeddedName(field.Type); n != "" {
		return []string{n}
	}
	return nil
}

// embeddedName возвращает имя встроенного поля по его типу.
func embeddedName(typ ast.Expr) string {
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return embeddedName(t.X)
	}
	return ""
}

// hasMarker сообщает, содержит ли группа комментариев маркер generate:reset.
func hasMarker(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, line := range strings.Split(cg.Text(), "\n") {
		if strings.TrimSpace(line) == marker {
			return true
		}
	}
	return false
}

// isResetMethod сообщает, является ли объявление методом Reset() без
// параметров и результатов.
func isResetMethod(d *ast.FuncDecl) bool {
	if d.Recv == nil || d.Name.Name != "Reset" {
		return false
	}
	if d.Type.Params != nil && len(d.Type.Params.List) > 0 {
		return false
	}
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		return false
	}
	return true
}

// receiverType возвращает имя базового типа получателя метода.
func receiverType(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	return embeddedName(d.Recv.List[0].Type)
}

// receiverName подбирает имя получателя по имени типа.
func receiverName(typeName string) string {
	for _, r := range typeName {
		return strings.ToLower(string(r))
	}
	return "r"
}

// sortedKeys возвращает отсортированные ключи map для детерминированного обхода.
func sortedKeys(m map[string][]*ast.File) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// isBasicType сообщает, является ли name предобъявленным примитивным типом.
func isBasicType(name string) bool {
	switch name {
	case "bool", "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune",
		"float32", "float64",
		"complex64", "complex128":
		return true
	}
	return false
}

// zeroValue возвращает литерал нулевого значения примитивного типа.
func zeroValue(name string) string {
	switch name {
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		return "0"
	}
}
