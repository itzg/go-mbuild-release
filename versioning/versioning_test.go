package versioning

import (
	"github.com/stretchr/testify/assert"
	"path"
	"testing"
)

func TestGitVersioning_GetVersion(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{
			path: "headIsTag", want: "2", wantErr: false,
		},
		{
			path: "relativeTag", want: "1-2-ff97468", wantErr: false,
		},
		{
			path: "noTag", want: "4b95fac", wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gv, err := NewGitVersioning(path.Join("testdata", tt.path))
			assert.NoError(t, err)

			got, err := gv.GetVersion()
			if tt.wantErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
