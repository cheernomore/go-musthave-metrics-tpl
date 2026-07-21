package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exampleSrc = `package example

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
	b     bool
}

// Ignored не помечена маркером и не должна получить Reset().
type Ignored struct {
	X int
}
`

func writePkg(t *testing.T, root, pkg, src string) string {
	t.Helper()
	dir := filepath.Join(root, pkg)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644))
	return dir
}

func TestGenerate(t *testing.T) {
	root := t.TempDir()
	pkgDir := writePkg(t, root, "example", exampleSrc)

	require.NoError(t, generate(root))

	genPath := filepath.Join(pkgDir, generatedFile)
	data, err := os.ReadFile(genPath)
	require.NoError(t, err)
	out := string(data)

	// Сгенерированный код должен быть синтаксически корректным Go.
	_, err = parser.ParseFile(token.NewFileSet(), genPath, data, parser.AllErrors)
	require.NoError(t, err)

	// Правила сброса из ТЗ.
	assert.Contains(t, out, "func (r *ResetableStruct) Reset() {")
	assert.Contains(t, out, "if r == nil {")
	assert.Contains(t, out, "r.i = 0")
	assert.Contains(t, out, `r.str = ""`)
	assert.Contains(t, out, "if r.strP != nil {")
	assert.Contains(t, out, `*r.strP = ""`)
	assert.Contains(t, out, "r.s = r.s[:0]")
	assert.Contains(t, out, "clear(r.m)")
	assert.Contains(t, out, "r.child.Reset()")
	assert.Contains(t, out, "r.b = false")

	// Непомеченная структура не упоминается.
	assert.NotContains(t, out, "Ignored")
	// Заголовок сгенерированного файла присутствует.
	assert.Contains(t, out, "Code generated")
}

// wideSrc покрывает все поддерживаемые виды полей: именованные типы,
// встроенные поля, массивы, интерфейсы, каналы, внешние типы и указатели.
const wideSrc = `package example

import "time"

// Inner имеет собственный метод Reset(), объявленный вручную.
type Inner struct{ N int }

func (i *Inner) Reset() { i.N = 0 }

type Celsius float64
type Names []string
type Dict map[string]int
type Plain struct{ X int }

// generate:reset
type Wide struct {
	Inner    Inner
	InnerP   *Inner
	Temp     Celsius
	TempP    *Celsius
	List     Names
	D        Dict
	P        Plain
	PP       *Plain
	Arr      [3]int
	SliceP   *[]int
	MapP     *map[string]int
	Anything any
	Err      error
	Ts       time.Time
	Ch       chan int
	Fn       func()
	Plain
}
`

func TestGenerate_AllTypeKinds(t *testing.T) {
	root := t.TempDir()
	pkgDir := writePkg(t, root, "example", wideSrc)

	require.NoError(t, generate(root))

	genPath := filepath.Join(pkgDir, generatedFile)
	data, err := os.ReadFile(genPath)
	require.NoError(t, err)
	out := string(data)

	_, err = parser.ParseFile(token.NewFileSet(), genPath, data, parser.AllErrors)
	require.NoError(t, err)

	want := []string{
		"func (w *Wide) Reset() {",
		"w.Inner.Reset()",             // значение с Reset
		"if w.InnerP != nil {",        // указатель с Reset
		"w.InnerP.Reset()",            //
		"w.Temp = 0",                  // именованный базовый
		"*w.TempP = 0",                // указатель на именованный базовый
		"w.List = w.List[:0]",         // именованный слайс
		"clear(w.D)",                  // именованная мапа
		"w.P = Plain{}",               // структура без Reset
		"*w.PP = Plain{}",             // указатель на структуру без Reset
		"w.Arr = [3]int{}",            // массив
		"*w.SliceP = (*w.SliceP)[:0]", // указатель на слайс
		"clear(*w.MapP)",              // указатель на мапу
		"w.Anything = nil",            // интерфейс
		"w.Err = nil",                 // error
		"w.Ts = time.Time{}",          // внешний тип
		"w.Ch = nil",                  // канал
		"w.Fn = nil",                  // функция
		"w.Plain = Plain{}",           // встроенное поле
		`"time"`,                      // импорт добавлен goimports
	}
	for _, s := range want {
		assert.Contains(t, out, s)
	}

	// Для Inner метод не генерируется: он объявлен вручную.
	assert.NotContains(t, out, "func (i *Inner) Reset()")
}

func TestGenerate_NoMarkers(t *testing.T) {
	root := t.TempDir()
	pkgDir := writePkg(t, root, "plain", "package plain\n\ntype T struct{ X int }\n")

	require.NoError(t, generate(root))

	_, err := os.Stat(filepath.Join(pkgDir, generatedFile))
	assert.True(t, os.IsNotExist(err), "без маркеров файл не должен создаваться")
}

func TestGenerate_SkipsTestdata(t *testing.T) {
	root := t.TempDir()
	pkgDir := writePkg(t, root, "testdata", exampleSrc)

	require.NoError(t, generate(root))

	_, err := os.Stat(filepath.Join(pkgDir, generatedFile))
	assert.True(t, os.IsNotExist(err), "каталог testdata должен пропускаться")
}

func TestHasMarker(t *testing.T) {
	assert.False(t, hasMarker(nil))
}

func TestZeroValue(t *testing.T) {
	assert.Equal(t, `""`, zeroValue("string"))
	assert.Equal(t, "false", zeroValue("bool"))
	assert.Equal(t, "0", zeroValue("int"))
	assert.Equal(t, "0", zeroValue("float64"))
}

func TestReceiverName(t *testing.T) {
	assert.Equal(t, "m", receiverName("Metrics"))
	assert.Equal(t, "r", receiverName(""))
}

func TestIsBasicType(t *testing.T) {
	assert.True(t, isBasicType("int"))
	assert.True(t, isBasicType("string"))
	assert.True(t, isBasicType("float64"))
	assert.False(t, isBasicType("MyStruct"))
}
