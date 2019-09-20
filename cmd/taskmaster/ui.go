package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
)

const (
	lw = 40

	ih = 3
)

func setKeyBindings(procs ProcessMap, p ProcChans, g *gocui.Gui) {
	//keybind
	err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	if err != nil {
		logger.Println("Could not set key binding:", err)
		return
	}
	fnk := wrap(&procs, p)
	err = g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, fnk)
	if err != nil {
		logger.Println("Cannot bind the enter key:", err)
	}
}

func runUI(procs ProcessMap, p ProcChans) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		logger.Println("Failed to create a GUI:", err)
		return
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(layout)
	setKeyBindings(procs, p, g)

	tw, th := g.Size()
	//list of process view
	{
		lv, err := g.SetView("list", 0, 0, lw, th-1)

		if err != nil && err != gocui.ErrUnknownView {
			logger.Println("Failed to create main view:", err)
			return
		}
		lv.Title = "List"
		lv.FgColor = gocui.ColorCyan
	}
	//output view
	{
		ov, err := g.SetView("output", lw+1, 0, tw-1, th-ih-1)
		if err != nil && err != gocui.ErrUnknownView {
			logger.Println("Failed to create output view:", err)
			return
		}
		ov.Title = "Output"
		ov.FgColor = gocui.ColorGreen

		ov.Autoscroll = true
		_, err = fmt.Fprintln(ov, "Press Ctrl-c to quit")
		if err != nil {
			logger.Println("Failed to print into output view:", err)
		}
	}
	//input view
	{
		iv, err := g.SetView("input", lw+1, th-ih, tw-1, th-1)
		if err != nil && err != gocui.ErrUnknownView {
			logger.Println("Failed to create input view:", err)
			return
		}
		iv.Title = "Input"
		iv.FgColor = gocui.ColorYellow

		iv.Editable = true
		err = iv.SetCursor(0, 0)
		if err != nil {
			logger.Println("Failed to set cursor:", err)
			return
		}
	}

	go updateStatusView(g, &procs)

	_, err = g.SetCurrentView("input")
	if err != nil {
		logger.Println("Cannot set focus to input view:", err)
	}

	err = g.MainLoop()
	logger.Println("Main loop has finished:", err)
}

func updateStatusView(g *gocui.Gui, procs *ProcessMap) {
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			g.Update(func(g *gocui.Gui) error {
				v, err := g.View("list")
				if err != nil {
					return err
				}
				v.Clear()
				fmt.Fprintln(v, procs)
				return nil
			})
		}
	}
}

func handleCommand(args []string, procs *ProcessMap, p ProcChans, f func(tmp []*Process, index int)) {
	if len(args) > 2 {
		if tmp, ok := (*procs)[args[1]]; ok {
			index, err := strconv.Atoi(args[2])
			if err != nil {
				logger.Println("atoi fialed", err)
				return
			}
			f(tmp, index)
		}
	}
}

func getCommand(line string, procs *ProcessMap, p ProcChans, ov *gocui.View) {
	args := strings.Fields(line)
	if len(args) > 0 {
		switch args[0] {
		case "clear":
			ov.Clear()
		case "status":
			handleCommand(args, procs, p, func(tmp []*Process, index int) {
				_, e := fmt.Fprint(ov, tmp[index].FullStatusString())
				if e != nil {
					logger.Println("Cannot print to output view:", e)
				}
			})
		case "start", "run":
			handleCommand(args, procs, p, func(tmp []*Process, index int) {
				p.newPros <- tmp[index]
			})
		case "stop":
			handleCommand(args, procs, p, func(tmp []*Process, index int) {
				p.oldPros <- tmp[index]
			})
		case "reload":
			*procs = UpdateConfig(configFile, *procs, p)
		case "help":
			ov.Clear()
			help := `Commands:
			status
				- Shows the status of a processes
			start, run
				- 'start [process name] [index]'   will start the process
			stop
				- 'stop [process name] [index]'   will stop the process
			reload
				- will reload the config file
			help
				- will show this prompt again
			clear
				- will clear the screen`
			_, e := fmt.Fprint(ov, help)
			if e != nil {
				logger.Println("Cannot print to output view:", e)
			}
		}
	}
}
func wrap(procs *ProcessMap, p ProcChans) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		iv, e := g.View("input")
		if e != nil {
			logger.Println("Cannot get output view:", e)
			return e
		}

		iv.Rewind()

		ov, e := g.View("output")
		if e != nil {
			logger.Println("Cannot get output view:", e)
			return e
		}
		line := iv.Buffer()
		getCommand(line, procs, p, ov)

		_, e = fmt.Fprint(ov, iv.Buffer())
		if e != nil {
			logger.Println("Cannot print to output view:", e)
		}

		iv.Clear()

		e = iv.SetCursor(0, 0)
		if e != nil {
			logger.Println("Failed to set cursor:", e)
		}
		return e
	}
}

func layout(g *gocui.Gui) error {

	tw, th := g.Size()

	_, err := g.SetView("list", 0, 0, lw, th-1)
	if err != nil {
		return errors.Wrap(err, "Cannot update list view")
	}
	_, err = g.SetView("output", lw+1, 0, tw-1, th-ih-1)
	if err != nil {
		return errors.Wrap(err, "Cannot update output view")
	}
	_, err = g.SetView("input", lw+1, th-ih, tw-1, th-1)
	if err != nil {
		return errors.Wrap(err, "Cannot update input view.")
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
