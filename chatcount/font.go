package chatcount

import (
	"os"
	"path/filepath"
)

func (p *PluginChatCount) getFontData() ([]byte, error) {
	return os.ReadFile(filepath.Join(p.filePath, p.conf.Font))
}
