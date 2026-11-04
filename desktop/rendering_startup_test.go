package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

func TestAutomaticRenderingPlan(t *testing.T) {
	for _, test := range []struct {
		name               string
		env                renderingEnvironment
		software, xwayland bool
	}{
		{"normal Wayland", renderingEnvironment{wayland: true, display: ":0"}, false, false},
		{"normal X11", renderingEnvironment{display: ":0"}, false, false},
		{"NVIDIA Wayland", renderingEnvironment{nvidia: true, wayland: true, display: ":1"}, false, true},
		{"NVIDIA without XWayland", renderingEnvironment{nvidia: true, wayland: true}, true, false},
		{"NVIDIA X11", renderingEnvironment{nvidia: true, display: ":0"}, false, false},
		{"explicit backend preserved", renderingEnvironment{nvidia: true, wayland: true, display: ":1", backend: "wayland"}, false, false},
		{"software recovery", renderingEnvironment{nvidia: true, wayland: true, display: ":1", software: true}, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := selectRenderingPlan(test.env)
			wantPolicy := linux.WebviewGpuPolicyOnDemand
			if test.software {
				wantPolicy = linux.WebviewGpuPolicyNever
			}
			if got.policy != wantPolicy || got.xwayland != test.xwayland {
				t.Fatalf("plan = %+v", got)
			}
			if (test.software || test.xwayland) && got.note == "" {
				t.Fatal("compatibility mode must be disclosed")
			}
		})
	}
}

func TestSoftwareRecoveryDoesNotSetGraphicsOverrides(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux startup environment")
	}
	t.Setenv("GDK_BACKEND", "wayland")
	t.Setenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM", "0")
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	plan := configureRendering([]string{"--software-rendering"})
	if plan.policy != linux.WebviewGpuPolicyNever {
		t.Fatal("software recovery flag ignored")
	}
	if os.Getenv("GDK_BACKEND") != "wayland" || os.Getenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM") != "0" {
		t.Fatal("explicit environment was overwritten")
	}
}

func TestNvidiaWorkaroundKeepsGPUAllowedAndDoesNotDisableDMABufRenderer(t *testing.T) {
	t.Setenv("GDK_BACKEND", "")
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	t.Setenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM", "")
	_ = os.Unsetenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM")
	plan := applyRenderingEnvironment(selectRenderingPlan(renderingEnvironment{nvidia: true, wayland: true, display: ":0"}))
	if plan.policy != linux.WebviewGpuPolicyOnDemand || os.Getenv("GDK_BACKEND") != "x11" || os.Getenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM") != "1" {
		t.Fatalf("GPU-compatible startup not configured: %+v", plan)
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") != "0" {
		t.Fatal("workaround must not disable the accelerated renderer")
	}
}

func TestExplicitSharedMemoryOverrideIsPreserved(t *testing.T) {
	t.Setenv("GDK_BACKEND", "")
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	t.Setenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM", "0")
	_ = applyRenderingEnvironment(selectRenderingPlan(renderingEnvironment{nvidia: true, wayland: true, display: ":0"}))
	if os.Getenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM") != "0" {
		t.Fatal("explicit override was changed")
	}
}
