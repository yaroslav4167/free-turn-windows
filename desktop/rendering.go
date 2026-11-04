package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

type renderingEnvironment struct {
	nvidia   bool
	wayland  bool
	display  string
	backend  string
	software bool
}

type renderingPlan struct {
	policy   linux.WebviewGpuPolicy
	xwayland bool
	note     string
}

func selectRenderingPlan(env renderingEnvironment) renderingPlan {
	plan := renderingPlan{policy: linux.WebviewGpuPolicyOnDemand}
	if env.software {
		plan.policy = linux.WebviewGpuPolicyNever
		plan.note = "Безопасный программный режим включён флагом --software-rendering."
		return plan
	}
	// Respect an explicit backend selected by the user. Never change DISPLAY or
	// invent an X server when the session has no XWayland endpoint.
	if !env.nvidia || !env.wayland || env.backend != "" {
		return plan
	}
	if env.display == "" {
		plan.policy = linux.WebviewGpuPolicyNever
		plan.note = "NVIDIA/Wayland: XWayland не предоставлен сеансом (DISPLAY не задан), поэтому используется безопасная программная отрисовка."
		return plan
	}
	plan.xwayland = true
	plan.note = "NVIDIA/Wayland: используется совместимый запуск через XWayland с разрешённым GPU-ускорением. Передача кадров через общую память может добавлять копирование; фактический рендерер выбирает WebKit/драйвер."
	return plan
}

func nvidiaDriverPresent() bool {
	for _, filename := range []string{"/proc/driver/nvidia/version", "/sys/module/nvidia/version"} {
		if _, err := os.Stat(filename); err == nil {
			return true
		}
	}
	return false
}

// Configure graphics BEFORE GTK/WebKit create a display or a webview. This
// changes only this process's environment, never the desktop session/config.
func configureRendering(args []string) renderingPlan {
	if runtime.GOOS != "linux" {
		return renderingPlan{policy: linux.WebviewGpuPolicyOnDemand}
	}
	session := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	plan := selectRenderingPlan(renderingEnvironment{
		nvidia:  nvidiaDriverPresent(),
		wayland: session == "wayland" || (session == "" && os.Getenv("WAYLAND_DISPLAY") != ""),
		display: os.Getenv("DISPLAY"), backend: os.Getenv("GDK_BACKEND"),
		software: slices.Contains(args, "--software-rendering"),
	})
	return applyRenderingEnvironment(plan)
}

func applyRenderingEnvironment(plan renderingPlan) renderingPlan {
	if plan.xwayland {
		_ = os.Setenv("GDK_BACKEND", "x11")
		// Unlike WEBKIT_DISABLE_DMABUF_RENDERER, FORCE_SHM keeps the accelerated
		// renderer eligible, but avoids direct DMA-BUF import into the GTK surface.
		if _, explicit := os.LookupEnv("WEBKIT_DMABUF_RENDERER_FORCE_SHM"); !explicit {
			_ = os.Setenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM", "1")
		}
	}
	if disabled := os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"); disabled != "" && disabled != "0" {
		plan.note += " В окружении задан WEBKIT_DISABLE_DMABUF_RENDERER: аппаратный композитинг может быть отключён независимо от политики приложения."
	}
	if plan.note != "" {
		fmt.Fprintln(os.Stderr, "FreeTurn graphics:", plan.note)
	}
	return plan
}
