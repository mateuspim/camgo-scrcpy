package main

import (
	"os"
	"testing"
)

func TestCheckDependencies(t *testing.T) {
	deps := checkDependencies()
	if deps == nil {
		t.Log("All dependencies found")
	}
}

func TestGetAdbDevices(t *testing.T) {
	devices := getAdbDevices()
	if devices == nil {
		t.Log("No devices found (expected if none connected)")
	}
}

func TestResolutionStruct(t *testing.T) {
	resolutions := []Resolution{
		{"Nativa (Máxima Qualidade)", 0},
		{"Full HD (1920px) - Padrão", 1920},
		{"HD (1280px) - Baixa Latência", 1280},
		{"SD (800px) - Wi-Fi Lento", 800},
	}

	if len(resolutions) != 4 {
		t.Errorf("Expected 4 resolution options, got %d", len(resolutions))
	}

	if resolutions[0].Value != 0 {
		t.Errorf("Expected native resolution to be 0, got %d", resolutions[0].Value)
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()

	if m.state != 0 {
		t.Errorf("Expected initial state to be 0, got %d", m.state)
	}

	if len(m.resolutions) != 4 {
		t.Errorf("Expected 4 resolution options, got %d", len(m.resolutions))
	}
}

func TestVideoDevicePath(t *testing.T) {
	videoDevice := "/dev/video10"
	if _, err := os.Stat(videoDevice); os.IsNotExist(err) {
		t.Log("video10 not found, will fallback to video0")
	}
}

func TestRenderOption(t *testing.T) {
	selected := renderOption("test", true)
	notSelected := renderOption("test", false)

	if selected == "" {
		t.Error("Selected option should not be empty")
	}

	if notSelected == "" {
		t.Error("Non-selected option should not be empty")
	}
}
