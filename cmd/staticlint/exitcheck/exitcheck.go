// Package exitcheck содержит анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main.
//
// Прямой вызов os.Exit в main завершает программу немедленно, минуя
// отложенные вызовы (defer) и корректное освобождение ресурсов. Вместо него
// следует возвращать ошибку из вспомогательной функции (например, run) и
// обрабатывать её в одном месте.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer запрещает прямой вызов os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Анализатор применим только к пакету main.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Пропускаем сгенерированный код (в частности, синтетический
		// _testmain.go, в котором go test вызывает os.Exit(m.Run())).
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			// Интересует только функция main без получателя.
			if !ok || fn.Recv != nil || fn.Name.Name != "main" {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isOSExit(pass, call) {
					pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main запрещён")
				}
				return true
			})
		}
	}

	return nil, nil
}

// isOSExit сообщает, является ли вызов обращением к os.Exit. Проверка опирается
// на информацию о типах, поэтому переопределённый локально идентификатор os
// не вызовет ложного срабатывания.
func isOSExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Exit" {
		return false
	}

	obj := pass.TypesInfo.ObjectOf(sel.Sel)
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "os" && obj.Name() == "Exit"
}
