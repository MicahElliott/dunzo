package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// compactTheme wraps another fyne.Theme (LightTheme, per
// BuildMainWindow's forced light-mode -- see eod.go's comment on why
// LightTheme is forced), only overriding the spacing sizes that drive
// "how spacey everything looks": Padding (the gap VBox/HBox/etc. put
// between sibling widgets -- this is what makes bulleted/checklist
// lists and stacked sections feel airy), InnerPadding (the padding
// widgets like Button put around their own content -- shrinking this
// is what "shrink buttons whenever possible" asks for, since a
// button's height is largely driven by this value), and LineSpacing
// (gap between wrapped text lines within a single Label/RichText).
// Text size and everything else (colors, icons, radii) are left at
// the wrapped theme's defaults -- this is purely a density tweak, not
// a full custom theme.
//
// IMPORTANT: this must be applied *after* (or instead of) any other
// a.Settings().SetTheme call, since SetTheme fully replaces whatever
// was set before -- BuildMainWindow's own
// a.Settings().SetTheme(theme.LightTheme()) previously silently
// clobbered this theme when it was set earlier in MakeUI, which was
// the actual reason two earlier tightening passes (2026-09-02)
// appeared to have no visible effect at all -- the compact theme was
// never actually active. Fixed by having newCompactTheme wrap
// LightTheme directly and having BuildMainWindow set *this* theme
// instead of a bare LightTheme.
//
// Default values (theme/size.go): Padding=4, InnerPadding=8,
// LineSpacing=4. Currently set to Padding=1, InnerPadding=2,
// LineSpacing=1 -- effectively floor values (can't go meaningfully
// below 0-1 without widgets visually touching/overlapping).
type compactTheme struct {
	fyne.Theme
}

func newCompactTheme() fyne.Theme {
	return compactTheme{Theme: theme.LightTheme()}
}

func (compactTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 1
	case theme.SizeNameInnerPadding:
		return 2
	case theme.SizeNameLineSpacing:
		return 1
	default:
		return theme.LightTheme().Size(name)
	}
}
