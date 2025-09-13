package xiaohongshu

import (
	"context"
	"testing"

	"github.com/xpzouying/xiaohongshu-mcp/browser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish(t *testing.T) {

	t.Skip("SKIP: 测试发布")

	b := browser.NewBrowser(false)
	// Browser will remain open - no Close() call

	page := b.NewPage()
	// Page will remain open - no Close() call

	action, err := NewPublishImageAction(page)
	require.NoError(t, err)

	err = action.Publish(context.Background(), PublishImageContent{
		Title:      "Hello World",
		Content:    "Hello World",
		ImagePaths: []string{"/tmp/1.jpg"},
	})
	assert.NoError(t, err)
}
