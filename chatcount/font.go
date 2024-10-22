package chatcount

import "os"

func (p *PluginChatCount) getFontData() ([]byte, error) {
	path := p.conf.Font
	return os.ReadFile(path)
}
