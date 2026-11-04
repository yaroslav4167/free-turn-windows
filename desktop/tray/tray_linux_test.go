//go:build linux

package tray

import (
	"reflect"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestMenuLayout(t *testing.T) {
	menu := &trayMenu{}
	revision, root, err := menu.GetLayout(0, -1, nil)
	if err != nil || revision != 1 || root.ID != 0 || len(root.Children) != 2 {
		t.Fatalf("invalid root layout: %+v, %v", root, err)
	}
	if got := dbus.SignatureOf(root).String(); got != "(ia{sv}av)" {
		t.Fatalf("layout signature = %s", got)
	}
	for index, child := range root.Children {
		item := child.Value().(menuLayout)
		if item.ID != int32(index+1) || item.Properties["enabled"].Value() != true {
			t.Fatalf("invalid menu child: %+v", item)
		}
	}
	_, shallow, err := menu.GetLayout(0, 0, nil)
	if err != nil || len(shallow.Children) != 0 {
		t.Fatalf("invalid shallow layout: %+v, %v", shallow, err)
	}
	if _, _, err := menu.GetLayout(99, -1, nil); err == nil {
		t.Fatal("unknown menu ID accepted")
	}
}

func TestMenuCallbacks(t *testing.T) {
	show, quit := make(chan struct{}, 1), make(chan struct{}, 1)
	menu := &trayMenu{&Tray{onShow: func() { show <- struct{}{} }, onQuit: func() { quit <- struct{}{} }}}
	for id, event := range map[int32]<-chan struct{}{1: show, 2: quit} {
		if err := menu.Event(id, "clicked", dbus.MakeVariant(0), 0); err != nil {
			t.Fatal(err)
		}
		select {
		case <-event:
		case <-time.After(time.Second):
			t.Fatalf("menu callback %d not invoked", id)
		}
	}
	invalid, err := menu.EventGroup([]menuEvent{{ID: 99, Event: "clicked", Data: dbus.MakeVariant(0)}})
	if err != nil || !reflect.DeepEqual(invalid, []int32{99}) {
		t.Fatalf("invalid IDs = %v, %v", invalid, err)
	}
}

func TestNoTrayIsUnavailable(t *testing.T) {
	var tray *Tray
	if tray.Available() {
		t.Fatal("nil tray is available")
	}
}

func TestApplicationIconPixmap(t *testing.T) {
	icons, err := applicationIcon(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(icons) != 1 || icons[0].Width != 64 || icons[0].Height != 64 || len(icons[0].Pixels) != 64*64*4 {
		t.Fatal("invalid application icon dimensions")
	}
	if got := dbus.SignatureOf(icons).String(); got != "a(iiay)" {
		t.Fatalf("icon signature = %s", got)
	}
}
