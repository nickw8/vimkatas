package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/examples"
	"github.com/gcla/gowid/widgets/button"
	"github.com/gcla/gowid/widgets/clicktracker"
	"github.com/gcla/gowid/widgets/columns"
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
//============ var ==========================================================

var app *gowid.App
var cols *ResizeableColumnsWidget
var rows1 *ResizeablePileWidget
var rows2 *ResizeablePileWidget
var vimWidget *terminal.Widget
var viewHolder *holder.Widget

//var controller *exerciseController
//var view *exerciseView
//var yesno *dialog.Widget
//var viewHolder *holder.Widget

//============ Resizable Columns Widget ==========================================================

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

//============ Resizable Pillar Widget ==========================================================

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


//============ Input Handler ==========================================================

type handler struct{}

func (h handler) UnhandledInput(app gowid.IApp, ev interface{}) bool {
	handled := false

	if evk, ok := ev.(*tcell.EventKey); ok {
		switch evk.Key() {
		case tcell.KeyEsc:
			handled = true
			vimWidget.Signal(syscall.SIGINT)
		case tcell.KeyCtrlBackslash, tcell.KeyCtrlC:
			handled = true
			vimWidget.Signal(syscall.SIGQUIT)
		//case tcell.KeyTAB:
		//	handled = true
		//	nextExercise()
		case tcell.KeyRune:
			handled = true
			switch evk.Rune() {
			case '>':
				cols.offset += 1
			case '<':
				cols.offset -= 1
			case '+':
				rows1.offset += 1
			case '-':
				rows1.offset -= 1
			default:
				handled = false
			}
		}
	}
	return handled
}

//============ Vim Widget ================================================

func makeNewVimWidget(fp string) *terminal.Widget {

	hkDuration := terminal.HotKeyDuration{D: time.Second * 3}
	v, err := terminal.NewExt(terminal.Options{
		Command:           strings.Split("vim" + " -R " + fp, " "),
		HotKeyPersistence: &hkDuration,
		Scrollback:        100,
	})
	if err != nil {
		panic(err)
	}
	return v
}

//============ Output Widget ================================================

func makeNewOutputWidget(content string) *text.Widget {
	o := text.NewFromContent(
		text.NewContent([]text.ContentSegment{
			text.StyledContent(content, gowid.MakePaletteRef("red")),
		}))
	return o
}
//============ Tips Widget ================================================

func makeNewTipsWidget(content string) *styled.Widget {
	t := styled.New(
		text.NewFromContentExt(
			text.NewContent([]text.ContentSegment{
				text.StyledContent(content, gowid.MakePaletteRef("banner")),
			}),
			text.Options{
				Align: gowid.HAlignLeft{},
			},
		),
		gowid.MakePaletteRef("streak"),
	)
	return t
}
//============ Menu Widget ================================================
type menuWidget struct {
	cols *ResizeableColumnsWidget
	nextBt *button.Widget
	exitBt *button.Widget

}
func makeNewMenuWidget() *menuWidget {
	p := gowid.RenderFixed{}
	nextText := text.New("Next")
	nextButton := button.New(nextText)

	quitText := text.New("Quit")
	quitButton := button.New(quitText)

	nextButtonStyled := styled.NewExt(nextButton,
		gowid.MakePaletteRef("button normal"),
		gowid.MakePaletteRef("button select"))
	quitButtonStyled := styled.NewExt(quitButton,
		gowid.MakePaletteRef("button normal"),
		gowid.MakePaletteRef("button select"))

	nextButtonTracker := clicktracker.New(nextButtonStyled)
	quitButtonTracker := clicktracker.New(quitButtonStyled)

	cols := NewResizeableColumns([]gowid.IContainerWidget{
		&gowid.ContainerWidget{hpadding.New(nextButtonTracker, gowid.HAlignMiddle{}, p), gowid.RenderWithWeight{1}},
		&gowid.ContainerWidget{hpadding.New(quitButtonTracker, gowid.HAlignMiddle{}, p), gowid.RenderWithWeight{1}},
	})

	res := &menuWidget{
		cols:   cols,
		nextBt: nextButton,
		exitBt: quitButton,
	}
	return res
}
//============ Exercise View ================================================
type exerciseView struct {
	viewHolder *holder.Widget
	view       *framed.Widget
	title      *text.Widget
	titleInv   *styled.Widget
	holder     *holder.Widget
	vimWidget  *terminal.Widget
	menuWidget *menuWidget
}

func makeNewExerciseView() (*exerciseView, error){

	getKata, err := handlers.SelectKata()
	kataTips := string(getKata.Tips)
	kataNum := getKata.Kata
	kataExample := string(getKata.Example)
	kataVim := getKata.VimText


	tw := text.New(" VimKatas ")
	twi := styled.New(tw, gowid.MakePaletteRef("invred"))
	twp := holder.New(tw)

	vimWidget := makeNewVimWidget(kataVim)
	outputWidget := makeNewOutputWidget(kataExample)
	tipsWidget := makeNewTipsWidget(kataTips)
	menuWidget := makeNewMenuWidget()

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

	menuFrame := framed.New(menuWidget.cols, framed.Options{
		Frame: framed.UnicodeFrame,
		Title: "Menu",
	})

	rows1 = NewResizeablePile([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: vimFrame, D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: outFrame, D: gowid.RenderWithWeight{W: 3}},
	})

	rows2 = NewResizeablePile([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: tipsFrame, D: gowid.RenderWithWeight{W: 5}},
		&gowid.ContainerWidget{IWidget: menuFrame, D: gowid.RenderWithWeight{W: 1}},
	})

	cols = NewResizeableColumns([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: rows1, D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: rows2, D: gowid.RenderWithWeight{W: 1}},

	})

	view := framed.New(cols, framed.Options{
		Frame:       framed.UnicodeFrame,
		TitleWidget: twp,
	})


	res := &exerciseView{
		view: view,
		title: tw,
		titleInv: twi,
		holder: twp,
		vimWidget: vimWidget,
		menuWidget: menuWidget,
	}

	return res, err
}
//============ Exercise Controller ================================================

type exerciseController struct {
	view *exerciseView
}

func makeNewExerciseController() (*exerciseController,error) {


	res := &exerciseController{nil}
	view, err := makeNewExerciseView()
	res.view = view

	// === Setting vim controls ===
	res.view.vimWidget.OnProcessExited(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			app.Quit()
		},
	})

	res.view.vimWidget.OnBell(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			res.view.holder.SetSubWidget(res.view.titleInv, app)
			timer := time.NewTimer(time.Millisecond * 800)
			go func() {
				<-timer.C
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					res.view.holder.SetSubWidget(res.view.title, app)
				}))
			}()
		},
	})

	res.view.vimWidget.OnSetTitle(gowid.WidgetCallback{Name: "cb",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			w2 := w.(*terminal.Widget)
			res.view.title.SetText(" "+w2.GetTitle()+" ", app)
		},
	})

	// === setting button controls ===
	res.view.menuWidget.nextBt.OnClick(gowid.WidgetCallback{"cb", func(app gowid.IApp, w gowid.IWidget) {
		view, _ := makeNewExerciseView()
		viewHolder.SetSubWidget(view.view, app)
	}})

	res.view.menuWidget.exitBt.OnClick(gowid.WidgetCallback{"cb", func(app gowid.IApp, w gowid.IWidget) {
		app.Quit()
	}})

	return res, err
}

//============ main ======================================================

func main() {
	var err error

	f := examples.RedirectLogger("terminal.log")
	defer f.Close()


	palette := gowid.Palette{
		"invred": gowid.MakePaletteEntry(gowid.ColorBlack, gowid.ColorRed),
		"line":   gowid.MakeStyledPaletteEntry(gowid.NewUrwidColor("black"), gowid.NewUrwidColor("light gray"), gowid.StyleBold),
	}

	controller, err := makeNewExerciseController()

	viewHolder = holder.New(controller.view.view)

	app, err = gowid.NewApp(gowid.AppArgs{
		View:    viewHolder,
		Palette: &palette,
		Log:     log.StandardLogger(),
	})

	examples.ExitOnErr(err)

	app.MainLoop(handler{})
}