package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/examples"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/dialog"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/holder"
	"github.com/gcla/gowid/widgets/hpadding"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/terminal"
	"github.com/gcla/gowid/widgets/text"
	"github.com/gdamore/tcell"
	log "github.com/sirupsen/logrus"
	"strings"
	"syscall"
	"time"
	"vimkatas/handlers"
)

//======================================================================

type ResizeableColumnsWidget struct {
	*columns.Widget
	offset int
}

func NewResizeableColumns(widgets []gowid.IContainerWidget) *ResizeableColumnsWidget {
	res := &ResizeableColumnsWidget{}
	res.Widget = columns.New(widgets)
	return res
}

func (w *ResizeableColumnsWidget) WidgetWidths(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp) []int {
	widths := w.Widget.WidgetWidths(size, focus, focusIdx, app)
	addme := w.offset
	if widths[0]+addme < 0 {
		addme = -widths[0]
	} else if widths[1]-addme < 0 {
		addme = widths[1]
	}
	widths[0] += addme
	widths[1] -= addme
	return widths
}

func (w *ResizeableColumnsWidget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return columns.Render(w, size, focus, app)
}

func (w *ResizeableColumnsWidget) RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp) []gowid.ICanvas {
	return columns.RenderSubWidgets(w, size, focus, focusIdx, app)
}

func (w *ResizeableColumnsWidget) RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp) []gowid.IRenderBox {
	return columns.RenderedSubWidgetsSizes(w, size, focus, focusIdx, app)
}

func (w *ResizeableColumnsWidget) SubWidgetSize(size gowid.IRenderSize, newX int, sub gowid.IWidget, dim gowid.IWidgetDimension) gowid.IRenderSize {
	return w.Widget.SubWidgetSize(size, newX, sub, dim)
}

//======================================================================

type ResizeablePileWidget struct {
	*pile.Widget
	offset int
}

func NewResizeablePile(widgets []gowid.IContainerWidget) *ResizeablePileWidget {
	res := &ResizeablePileWidget{}
	res.Widget = pile.New(widgets)
	return res
}

type PileAdjuster struct {
	widget    *ResizeablePileWidget
	origSizer pile.IPileBoxMaker
}

func (f PileAdjuster) MakeBox(w gowid.IWidget, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	adjustedSize := size
	var box gowid.RenderBox
	isbox := false
	switch s2 := size.(type) {
	case gowid.IRenderBox:
		box.C = s2.BoxColumns()
		box.R = s2.BoxRows()
		isbox = true
	}
	i := 0
	for ; i < len(f.widget.SubWidgets()); i++ {
		if w == f.widget.SubWidgets()[i] {
			break
		}
	}
	if i == len(f.widget.SubWidgets()) {
		panic("Unexpected pile state!")
	}
	if isbox {
		switch i {
		case 0:
			if box.R+f.widget.offset < 0 {
				f.widget.offset = -box.R
			}
			box.R += f.widget.offset
		case 2:
			if box.R-f.widget.offset < 0 {
				f.widget.offset = box.R
			}
			box.R -= f.widget.offset
		}
		adjustedSize = box
	}
	return f.origSizer.MakeBox(w, adjustedSize, focus, app)
}

func (w *ResizeablePileWidget) FindNextSelectable(dir gowid.Direction, wrap bool) (int, bool) {
	return gowid.FindNextSelectableFrom(w, w.Focus(), dir, wrap)
}

func (w *ResizeablePileWidget) UserInput(ev interface{}, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	return pile.UserInput(w, ev, size, focus, app)
}

func (w *ResizeablePileWidget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return pile.Render(w, size, focus, app)
}

func (w *ResizeablePileWidget) RenderedSubWidgetsSizes(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp) []gowid.IRenderBox {
	res, _ := pile.RenderedChildrenSizes(w, size, focus, focusIdx, app)
	return res
}

func (w *ResizeablePileWidget) RenderSubWidgets(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp) []gowid.ICanvas {
	return pile.RenderSubwidgets(w, size, focus, focusIdx, app)
}

func (w *ResizeablePileWidget) RenderBoxMaker(size gowid.IRenderSize, focus gowid.Selector, focusIdx int, app gowid.IApp, sizer pile.IPileBoxMaker) ([]gowid.IRenderBox, []gowid.IRenderSize) {
	x := &PileAdjuster{
		widget:    w,
		origSizer: sizer,
	}
	return pile.RenderBoxMaker(w, size, focus, focusIdx, app, x)
}

//======================================================================

var app *gowid.App
var cols *ResizeableColumnsWidget
var pilew *ResizeablePileWidget
var vimWidget *terminal.Widget
var yesno *dialog.Widget
var viewHolder *holder.Widget

//======================================================================

type handler struct{}

func (h handler) UnhandledInput(app gowid.IApp, ev interface{}) bool {
	handled := false

	if evk, ok := ev.(*tcell.EventKey); ok {
		switch evk.Key() {
		case tcell.KeyEsc:
			handled = true
			vimWidget.Signal(syscall.SIGINT)
		case tcell.KeyCtrlC:
			handled = true
			msg := text.New("Do you want to quit?")
			yesno = dialog.New(
				framed.NewSpace(hpadding.New(msg, gowid.HAlignMiddle{}, gowid.RenderFixed{})),
				dialog.Options{
					Buttons: dialog.OkCancel,
				},
			)
			yesno.Open(viewHolder, gowid.RenderWithRatio{R: 0.5}, app)
		case tcell.KeyCtrlBackslash:
			handled = true
			vimWidget.Signal(syscall.SIGQUIT)
		case tcell.KeyRune:
			handled = true
			switch evk.Rune() {
			case '>':
				cols.offset += 1
			case '<':
				cols.offset -= 1
			case '+':
				pilew.offset += 1
			case '-':
				pilew.offset -= 1
			default:
				handled = false
			}
		}
	}
	return handled
}

//======================================================================

func main() {
	var err error
	getKata, err := handlers.SelectKata()
	kataTips := string(getKata.Tips)
	kataNum := getKata.Kata
	kataExample := string(getKata.Example)
	kataVim := getKata.VimText

	f := examples.RedirectLogger("terminal.log")
	defer f.Close()

	palette := gowid.Palette{
		"invred": gowid.MakePaletteEntry(gowid.ColorBlack, gowid.ColorRed),
		"line":   gowid.MakeStyledPaletteEntry(gowid.NewUrwidColor("black"), gowid.NewUrwidColor("light gray"), gowid.StyleBold),
	}

	hkDuration := terminal.HotKeyDuration{D: time.Second * 3}

	tw := text.New(" VimKatas ")
	twi := styled.New(tw, gowid.MakePaletteRef("invred"))
	twp := holder.New(tw)

	vimWidget, err := terminal.NewExt(terminal.Options{
		Command:           strings.Split("vim" + " -R " + kataVim, " "),
		HotKeyPersistence: &hkDuration,
		Scrollback:        100,
		})
		if err != nil {
			panic(err)
	}

	vimWidget.OnProcessExited(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			app.Quit()
		},
	})

	vimWidget.OnBell(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			twp.SetSubWidget(twi, app)
			timer := time.NewTimer(time.Millisecond * 800)
			go func() {
				<-timer.C
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					twp.SetSubWidget(tw, app)
				}))
			}()
		},
	})

	vimWidget.OnSetTitle(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			w2 := w.(*terminal.Widget)
			tw.SetText(" "+w2.GetTitle()+" ", app)
		},
	})

	outputWidget := text.NewFromContent(
			text.NewContent([]text.ContentSegment{
				text.StyledContent(kataExample, gowid.MakePaletteRef("red")),
			}))

	tipsWidget := styled.New(
		text.NewFromContentExt(
			text.NewContent([]text.ContentSegment{
				text.StyledContent(kataTips, gowid.MakePaletteRef("banner")),
			}),
			text.Options{
				Align: gowid.HAlignLeft{},
			},
		),
		gowid.MakePaletteRef("streak"),
	)

	outFrame := framed.New(outputWidget, framed.Options{
		Frame: framed.UnicodeFrame,
		Title: "Expected Output",
	})

	tipsFrame := framed.New(tipsWidget, framed.Options{
		Frame: framed.UnicodeFrame,
		Title: "Tips",
	})

	vimFrame := framed.New(vimWidget, framed.Options{
		Frame: framed.UnicodeFrame,
		Title: "Exercise " + kataNum,
	})

	pilew = NewResizeablePile([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: vimFrame, D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: outFrame, D: gowid.RenderWithWeight{W: 3}},
	})

	cols = NewResizeableColumns([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: pilew, D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: tipsFrame, D: gowid.RenderWithWeight{W: 1}},

	})

	view := framed.New(cols, framed.Options{
		Frame:       framed.UnicodeFrame,
		TitleWidget: twp,
	})

	app, err = gowid.NewApp(gowid.AppArgs{
		View:    view,
		Palette: &palette,
		Log:     log.StandardLogger(),
	})
	examples.ExitOnErr(err)

	app.MainLoop(handler{})
}