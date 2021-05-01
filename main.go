package main

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gioui.org/font/gofont"
)

//go:embed message.txt
var s string
var url = "https://craftinggames.eu/StyyClient/StyyClient%20NEW.zip"
var limit = &layout.List{Axis: layout.Vertical}
var installBtn = new(widget.Clickable)
var running = false
var progress = float64(0)
var status = ""
var fatal error
var th = material.NewTheme(gofont.Collection())
var mcVersionPath = os.Getenv(`APPDATA`) + `\.minecraft\versions\StyyClient`

func main() {
	go func() {
		w := app.NewWindow(app.Title("Styy Client installer by Wizard_x"))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

var ops op.Ops

func loop(w *app.Window) error {
	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if fatal == nil {
				renderMain(gtx)
			} else {
				renderFatal(gtx, fatal)
			}
			e.Frame(gtx.Ops)
		}
	}
}

func renderMain(gtx layout.Context) {
	l := material.H3(th, "(Unofficial) Styy Client installer")
	maroon := color.NRGBA{R: 127, G: 0, B: 0, A: 255}
	l.Color = maroon
	l.Alignment = text.Middle

	l2 := material.Label(th, th.TextSize.Scale(1.5), s)
	l2.Alignment = text.Start

	statusLabel := material.Label(th, th.TextSize.Scale(1.5), status)
	statusLabel.Alignment = text.Start

	widgets := []layout.Widget{
		inset(gtx, l.Layout, unit.Dp(30)),
		inset(gtx, l2.Layout, unit.Dp(30)),
		inset(gtx, func(gtx layout.Context) layout.Dimensions {
			if running {
				gtx = gtx.Disabled()
			}
			return material.Button(th, installBtn, "Installieren").Layout(gtx)
		}, unit.Dp(30)),
		inset(gtx, statusLabel.Layout, unit.Dp(30)),
		inset(gtx, material.ProgressBar(th, float32(progress)).Layout, unit.Dp(30)),
	}

	for installBtn.Clicked() {
		if running {
			break
		}
		go downloadClient()
		running = true
	}

	if running {
		op.InvalidateOp{}.Add(&ops)
	}

	limit.Layout(gtx, len(widgets), func(gtx layout.Context, i int) layout.Dimensions {
		c := widgets[i]
		return c(gtx)
	})
}

func renderFatal(gtx layout.Context, fatal error) {
	l := material.H3(th, "Ein fehler ist aufgetreten!")
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	l.Color = red
	l.Alignment = text.Middle

	l2 := material.Label(th, th.TextSize.Scale(2), fatal.Error())
	l2.Alignment = text.Start
	l2.Color = red

	widgets := []layout.Widget{
		inset(gtx, l.Layout, unit.Dp(30)),
		inset(gtx, l2.Layout, unit.Dp(30)),
	}

	limit.Layout(gtx, len(widgets), func(gtx layout.Context, i int) layout.Dimensions {
		c := widgets[i]
		return c(gtx)
	})
}

func downloadClient() {
	status = "Downloading Client..."
	r, err := http.Get(url)
	if err != nil {
		fatal = err
		return
	}

	tmpD, err := os.MkdirTemp("", "styydevdownload_*")

	if err != nil {
		fatal = err
		return
	}

	tempF := filepath.Join(tmpD, "/client.zip")

	file, err := os.OpenFile(tempF, os.O_CREATE, 0o777)
	if err != nil {
		fatal = err
		return
	}
	wcnt := NewWriteCounter(r.ContentLength, file, func(f float64) { progress = f })
	err = wcnt.WriteFullFrom(r.Body)
	if err != nil {
		fatal = err
		return
	}
	status = "Decompressing client..."
	if err = Unzip(tempF, mcVersionPath); err != nil {
		fatal = err
		return
	}
	status = "Installation Abgeschlossen!"
	r.Body.Close()
	file.Close()
	running = false
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0o777)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			_f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o777)
			if err != nil {
				return err
			}
			defer func() {
				if err := _f.Close(); err != nil {
					panic(err)
				}
			}()
			wcnt := NewWriteCounter(f.FileInfo().Size(), _f, func(f float64) { progress = f })
			status = "Decompressing " + f.Name
			err = wcnt.WriteFullFrom(rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func inset(gtx layout.Context, l layout.Widget, ins unit.Value) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(ins).Layout(gtx, l)
	}
}
