package fusionbrain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_prepareRequestBody(t *testing.T) {
	tests := []struct {
		name           string
		style          string
		prompt         string
		negativePrompt string
		width          int
		height         int
		exp            string
	}{
		{
			name:           "Сборка тела запроса",
			style:          "DEFAULT",
			prompt:         "test prompt",
			negativePrompt: "test negative prompt",
			width:          1024,
			height:         1024,
			exp:            `{"type":"GENERATE","style":"DEFAULT","width":1024,"height":1024,"num_images":1,"negativePromptUnclip":"test negative prompt","generateParams":{"query":"test prompt"}}`,
		},
	}
	for _, tt := range tests {
		reqBody := RequestBody{
			Style:          tt.style,
			Prompt:         tt.prompt,
			NegativePrompt: tt.negativePrompt,
			Width:          tt.width,
			Height:         tt.height,
		}
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, string(reqBody.serialize()), tt.exp)
		})
	}
}
