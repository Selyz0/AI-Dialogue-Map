package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// --- カスタムテーマ定義 ---
type myTheme struct {
	fyne.Theme
}

func NewMyTheme() fyne.Theme {
	return &myTheme{Theme: theme.DarkTheme()}
}

func (m *myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return m.Theme.Font(style)
}

func (m *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	baseColor := m.Theme.Color(name, variant)
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x03, G: 0xa9, B: 0xf4, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x42, G: 0x42, B: 0x42, A: 0xff}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0x52, G: 0x52, B: 0x52, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x75, G: 0x75, B: 0x75, A: 0xff}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x03, G: 0xa9, B: 0xf4, A: 0x77}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x50}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x61, G: 0x61, B: 0x61, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x2c, G: 0x2c, B: 0x2c, A: 0xff}
	}
	return baseColor
}

func (m *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return m.Theme.Icon(name)
}

func (m *myTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 6
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInputBorder:
		return 1
	}
	return m.Theme.Size(name)
}
