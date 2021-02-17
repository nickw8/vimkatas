package main

import (
	"github.com/rivo/tview"
)


func main() {
	app := tview.NewApplication()
	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewBox().SetBorder(true).SetTitle("Legend"), 3, 1, false).
			AddItem(tview.NewBox().SetBorder(true).SetTitle("Vim Console"), 0, 3, true).
			AddItem(tview.NewBox().SetBorder(true).SetTitle("Expected Output"), 0, 3, false), 0, 2, false).
		AddItem(tview.NewBox().SetBorder(true).SetTitle("Tips"), 40, 1, false)
	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}