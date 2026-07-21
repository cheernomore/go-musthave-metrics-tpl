package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildChecks(t *testing.T) {
	checks := buildChecks()
	require.NotEmpty(t, checks)

	names := make(map[string]bool, len(checks))
	var saCount int
	for _, a := range checks {
		require.NotNil(t, a)
		// Имена анализаторов должны быть уникальны, иначе multichecker.Main
		// завершится паникой при старте.
		assert.Falsef(t, names[a.Name], "дублирующееся имя анализатора: %s", a.Name)
		names[a.Name] = true
		if strings.HasPrefix(a.Name, "SA") {
			saCount++
		}
	}

	// Должны присутствовать все ключевые группы анализаторов.
	assert.Positive(t, saCount, "ожидались анализаторы класса SA")
	assert.True(t, names["exitcheck"], "ожидался собственный анализатор exitcheck")
	assert.True(t, names["bodyclose"], "ожидался публичный анализатор bodyclose")
	assert.True(t, names["printf"], "ожидался стандартный анализатор printf")
}

func TestStandardPasses(t *testing.T) {
	passes := standardPasses()
	require.NotEmpty(t, passes)
	for _, a := range passes {
		assert.NotNil(t, a)
		assert.NotEmpty(t, a.Name)
	}
}
