package exitcheck_test

import (
	"testing"

	"github.com/cheernomore/go-musthave-metrics-tpl/cmd/staticlint/exitcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestExitCheckAnalyzer проверяет анализатор на наборе пакетов:
//   - a: прямой os.Exit в main (ожидается диагностика) и в helper (нет);
//   - b: пакет не main (нет диагностики);
//   - c: локально переопределённый os (нет диагностики).
func TestExitCheckAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exitcheck.Analyzer, "a", "b", "c")
}
