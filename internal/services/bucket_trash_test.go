package services

import (
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildFolderPath(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	foldersByID := map[uuid.UUID]models.Folder{
		parentID: {ID: parentID, Name: "parent"},
		childID:  {ID: childID, Name: "child", FolderID: &parentID},
	}

	assert.Equal(t, "/", buildFolderPath(nil, foldersByID))
	assert.Equal(t, "/parent/child", buildFolderPath(&childID, foldersByID))
	assert.Equal(t, "/", buildFolderPath(new(uuid.UUID), foldersByID))
}
