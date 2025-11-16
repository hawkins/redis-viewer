package keylist

import (
	"github.com/hawkins/redis-viewer/internal/styles"
)

// View renders the keylist component
func (m Model) View() string {
	style := styles.ListViewStyle
	if m.focused {
		style = style.BorderForeground(styles.ListFocusedBorderColor)
	} else {
		style = style.BorderForeground(styles.ListUnfocusedBorderColor)
	}
	return style.Render(m.list.View())
}
