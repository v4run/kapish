package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhaseGlyph(t *testing.T) {
	assert.Equal(t, "✓", phaseGlyph("Provisioned"))
	assert.Equal(t, "…", phaseGlyph("Provisioning"))
	assert.Equal(t, "…", phaseGlyph("Pending"))
	assert.Equal(t, "⚠", phaseGlyph("Failed"))
	assert.Equal(t, "⚠", phaseGlyph("Deleting"))
	assert.Equal(t, "·", phaseGlyph(""))
	assert.Equal(t, "·", phaseGlyph("Weird"))
}

func TestPhaseStyledNotEmpty(t *testing.T) {
	out := phaseStyled("Provisioned")
	assert.Contains(t, out, "Provisioned")
}
