package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetEnvironmentVariableForLocalOCMTar(t *testing.T) {
	tests := []struct {
		name    string
		path    Path
		wantErr bool
	}{
		{
			name:    "Set valid path",
			path:    LocalOCMRepositoryPathValid,
			wantErr: false,
		},
		{
			name:    "Set invalid path",
			path:    RepositoryPathInvalid,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, SetEnvironmentVariableForLocalOCMTar(tt.path))
			if os.Getenv(OCMRepositoryPathKey) != string(tt.path) {
				t.Error("Environment variable not set correctly")
			}
		})
	}
}
