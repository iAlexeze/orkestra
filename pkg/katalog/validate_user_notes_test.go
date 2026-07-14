package katalog

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithNotes(reg orktypes.NoteRegistry) *Katalog {
	return &Katalog{
		Notes:       reg,
		enabledCRDs: map[string]orktypes.CRDEntry{},
	}
}

func TestValidateUserNotes_Empty(t *testing.T) {
	k := katalogWithNotes(nil)
	assert.NoError(t, k.validateUserNotes())
}

func TestValidateUserNotes_Valid(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "appImage", Expression: `{{ .spec.image }}`},
		{Name: "fullName", Expression: `{{ .metadata.namespace }}-{{ .metadata.name }}`},
	})
	assert.NoError(t, k.validateUserNotes())
}

func TestValidateUserNotes_MissingName(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "", Expression: `{{ .spec.image }}`},
	})
	err := k.validateUserNotes()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name must not be empty")
}

func TestValidateUserNotes_MissingExpression(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "myNote", Expression: ""},
	})
	err := k.validateUserNotes()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expression must not be empty")
}

func TestValidateUserNotes_Duplicate(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "myNote", Expression: `{{ .spec.image }}`},
		{Name: "myNote", Expression: `{{ .spec.replicas }}`},
	})
	err := k.validateUserNotes()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate note name")
}

func TestValidateUserNotes_InvalidTemplate(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "broken", Expression: `{{ .spec.image`},
	})
	err := k.validateUserNotes()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expression")
}

func TestValidateUserNotes_ShadowBuiltin_NoAck(t *testing.T) {
	// "default" is a built-in note — without shadow: true this warns but does not error
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "default", Expression: `{{ .spec.image }}`},
	})
	assert.NoError(t, k.validateUserNotes())
}

func TestValidateUserNotes_ShadowBuiltin_Acked(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "default", Expression: `{{ .spec.image }}`, Shadow: true},
	})
	assert.NoError(t, k.validateUserNotes())
}

func TestValidateUserNotes_WithDescription(t *testing.T) {
	k := katalogWithNotes(orktypes.NoteRegistry{
		{Name: "taggedImage", Description: "full image with tag", Expression: `{{ .spec.image }}:{{ .spec.tag }}`},
	})
	assert.NoError(t, k.validateUserNotes())
}
