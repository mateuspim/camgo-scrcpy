package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Estilos ---
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("205")).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("205")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Padding(1, 0)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

// --- Estruturas de Dados ---

type Resolution struct {
	Label string
	Value int
}

type scrcpyStartedMsg struct {
	cmd        *exec.Cmd
	err        error
	formatWarn string // non-empty if v4l2 format could not be pre-configured
}

type model struct {
	devices     []string
	resolutions []Resolution
	cursor      int

	// Seleções do usuário
	selectedDev      string
	selectedCam      string // "back" ou "front"
	selectedRotation string // capture-orientation value: "0", "90", "180", "270"
	selectedRes      int

	scrcpyProc  *exec.Cmd
	err         error
	formatWarn  string
	state       int // 0: Device, 1: Câmera, 2: Rotation (front only), 3: Resolução, 4: Rodando
	missingDeps []string
}

func initialModel() model {
	missing := checkDependencies()

	resOptions := []Resolution{
		{"Nativa (Máxima Qualidade)", 0},
		{"Full HD (1920px) - Padrão", 1920},
		{"HD (1280px) - Baixa Latência", 1280},
		{"SD (800px) - Wi-Fi Lento", 800},
	}

	devs := []string{}
	if len(missing) == 0 {
		devs = getAdbDevices()
	}

	return model{
		devices:     devs,
		resolutions: resOptions,
		state:       0,
		missingDeps: missing,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case scrcpyStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = 0
		} else {
			m.scrcpyProc = msg.cmd
			m.formatWarn = msg.formatWarn
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.scrcpyProc != nil {
				m.scrcpyProc.Process.Kill()
			}
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			limit := 0
			switch m.state {
			case 0:
				limit = len(m.devices) - 1
			case 1:
				limit = 1
			case 2:
				limit = 3 // 4 orientation options
			case 3:
				limit = len(m.currentResolutions()) - 1 // dynamic: portrait has 4, landscape has 4
			}
			if m.cursor < limit {
				m.cursor++
			}

		case "enter":
			switch m.state {
			case 0:
				if len(m.devices) > 0 {
					m.selectedDev = strings.Fields(m.devices[m.cursor])[0]
					m.state = 1
					m.cursor = 0
				}
			case 1:
				if m.cursor == 0 {
					m.selectedCam = "back"
					m.state = 3 // back camera: skip rotation menu
				} else {
					m.selectedCam = "front"
					m.state = 2 // front camera: show rotation menu
				}
				m.cursor = 0
			case 2:
				rotations := []string{"270", "90", "0", "180"}
				m.selectedRotation = rotations[m.cursor]
				m.state = 3
				m.cursor = 0 // portrait only has 2 options, start at first
			case 3:
				m.selectedRes = m.currentResolutions()[m.cursor].Value
				m.state = 4
				return m, startScrcpy(m.selectedDev, m.selectedCam, m.selectedRotation, m.selectedRes)
			}

		case "r":
			if m.state == 0 {
				m.devices = getAdbDevices()
			}
			if m.state == 4 && m.formatWarn != "" {
				oldDev := m.selectedDev
				oldCam := m.selectedCam
				oldRot := m.selectedRotation
				oldRes := m.selectedRes
				if m.scrcpyProc != nil {
					m.scrcpyProc.Process.Kill()
					m.scrcpyProc = nil
				}
				m.formatWarn = ""
				return m, startScrcpy(oldDev, oldCam, oldRot, oldRes)
			}
			if m.state == 4 {
				if m.scrcpyProc != nil {
					m.scrcpyProc.Process.Kill()
					m.scrcpyProc = nil
				}
				m.state = 0
				m.cursor = 0
				m.devices = getAdbDevices()
			}
		}

	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

// --- View ---

func (m model) View() string {
	s := "\n" + titleStyle.Render("🤖 Android Webcam TUI") + "\n\n"

	if len(m.missingDeps) > 0 {
		return renderError(s, m.missingDeps)
	}

	if m.err != nil {
		return fmt.Sprintf("Erro crítico: %v\nPressione q para sair.", m.err)
	}

	switch m.state {
	case 0:
		s += "Selecione o dispositivo (r para recarregar):\n\n"
		if len(m.devices) == 0 {
			s += itemStyle.Render("Nenhum dispositivo encontrado. Conecte via USB.")
		}
		for i, dev := range m.devices {
			s += renderOption(dev, m.cursor == i)
		}

	case 1:
		s += fmt.Sprintf("Dispositivo: %s\nEscolha a câmera:\n\n", m.selectedDev)
		for i, opt := range []string{"📸 Câmera Traseira", "🤳 Câmera Frontal"} {
			s += renderOption(opt, m.cursor == i)
		}

	case 2:
		s += fmt.Sprintf("Dispositivo: %s  •  Câmera: frontal\n", m.selectedDev)
		s += "Como você vai segurar o telefone?\n\n"
		opts := []string{
			"📱 Portrait — telefone em pé (90°)",
			"📱 Portrait invertido — em pé, girado (270°)",
			"🖥️  Landscape — telefone deitado (0°)",
			"🖥️  Landscape invertido — deitado, girado (180°)",
		}
		for i, opt := range opts {
			s += renderOption(opt, m.cursor == i)
		}

	case 3:
		s += fmt.Sprintf("Câmera: %s\nQualidade do stream (limite de tamanho):\n\n", m.selectedCam)
		for i, res := range m.currentResolutions() {
			s += renderOption(res.Label, m.cursor == i)
		}

	case 4:
		s += "🚀 STREAMING ATIVO\n\n"
		s += fmt.Sprintf("• Device: %s\n", m.selectedDev)
		s += fmt.Sprintf("• Câmera: %s\n", m.selectedCam)
		if m.selectedCam == "front" {
			s += fmt.Sprintf("• Orientação: %s°\n", m.selectedRotation)
		}
		resText := "Nativa"
		if m.selectedRes > 0 {
			resText = fmt.Sprintf("%dpx", m.selectedRes)
		}
		s += fmt.Sprintf("• Resolução Max: %s\n", resText)
		if m.formatWarn != "" {
			s += errorStyle.Render("\n⚠️  "+m.formatWarn) + "\n"
		}
		s += "\nMinimize esta janela e use sua webcam."
		s += helpStyle.Render("\n(r: reiniciar • Ctrl+C: encerrar)")
	}

	if m.state != 4 {
		s += helpStyle.Render("\n(enter: selecionar • q: sair)")
	}
	return s
}

// --- Helpers de Renderização ---

func isPortraitRotation(r string) bool {
	return r == "90" || r == "270"
}

func (m model) currentResolutions() []Resolution {
	if isPortraitRotation(m.selectedRotation) {
		// Portrait resolutions must avoid pixel counts equal to landscape equivalents
		// (e.g. 1080x1920 = 1920x1080 in total pixels → v4l2loopback reuses old buffer
		// stride → green blob). Use standard 16:9 portrait sizes with different totals.
		// Only sizes the front camera actually supports (landscape capture → rotated output)
		return []Resolution{
			{"Full HD (1080x1920)", 1920},
			{"HD (720x1280)", 1280},
			{"SD (360x640)", 640},
		}
	}
	return m.resolutions
}

func renderOption(text string, isSelected bool) string {
	if isSelected {
		return fmt.Sprintf("%s\n", selectedStyle.Render(text))
	}
	return fmt.Sprintf("%s\n", itemStyle.Render(text))
}

func renderError(base string, deps []string) string {
	msg := "⚠️  Faltam dependências:\n"
	for _, dep := range deps {
		msg += fmt.Sprintf("  • %s\n", dep)
	}
	msg += "\nExecute: sudo apt install adb scrcpy v4l2loopback-dkms"
	return base + errorStyle.Render(msg)
}

// --- Lógica de Sistema ---

func checkDependencies() []string {
	var missing []string
	if _, err := os.Stat("/sys/module/v4l2loopback"); os.IsNotExist(err) {
		missing = append(missing, "Driver v4l2loopback (não carregado)")
	}
	if _, err := exec.LookPath("adb"); err != nil {
		missing = append(missing, "adb")
	}
	if _, err := exec.LookPath("scrcpy"); err != nil {
		missing = append(missing, "scrcpy")
	}
	return missing
}

func getAdbDevices() []string {
	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return []string{}
	}

	lines := strings.Split(string(out), "\n")
	var devices []string
	for _, line := range lines {
		if strings.Contains(line, "device") && !strings.Contains(line, "List of") {
			devices = append(devices, line)
		}
	}
	return devices
}

func findLoopbackDevice() string {
	for i := 0; i < 64; i++ {
		dev := fmt.Sprintf("/dev/video%d", i)
		if _, err := os.Stat(dev); err != nil {
			continue
		}
		// Real hardware devices have a sysfs "device" symlink; v4l2loopback does not
		deviceLink := fmt.Sprintf("/sys/class/video4linux/video%d/device", i)
		if _, err := os.Stat(deviceLink); err == nil {
			continue
		}
		return dev
	}
	return ""
}

type camSize struct{ w, h int }

var knownCameraSizes = map[int]camSize{
	1920: {1920, 1080},
	1280: {1280, 720},
	800:  {800, 450},
	640:  {640, 360},
	0:    {1920, 1080}, // native: default to FHD
}

func startScrcpy(serial, facing, rotation string, resolution int) tea.Cmd {
	return func() tea.Msg {
		videoDevice := findLoopbackDevice()
		if videoDevice == "" {
			return scrcpyStartedMsg{err: fmt.Errorf("nenhum dispositivo v4l2loopback encontrado.\nExecute: sudo modprobe v4l2loopback")}
		}

		cs, ok := knownCameraSizes[resolution]
		if !ok {
			cs = camSize{1920, 1080}
		}

		// Output dimensions: portrait rotation swaps width/height
		outW, outH := cs.w, cs.h
		if isPortraitRotation(rotation) {
			outW, outH = cs.h, cs.w
		}

		// Pre-configure v4l2loopback with the exact output format so OBS
		// negotiates the correct resolution on open (without this, OBS can
		// latch onto a stale format from a previous session and produce garbled output).
		//
		// This fails with EBUSY when OBS (or any consumer) already has the device
		// open — in that case the portrait/landscape byte counts are identical
		// (e.g. 1080×1920 == 1920×1080 == 3,110,400 bytes) so v4l2loopback passes
		// the data through silently but OBS reads it with the wrong stride → garbled.
		var formatWarn string
		if err := exec.Command("v4l2-ctl", "--device="+videoDevice,
			fmt.Sprintf("--set-fmt-video=width=%d,height=%d,pixelformat=YU12", outW, outH),
		).Run(); err != nil {
			formatWarn = fmt.Sprintf(
				"Formato do dispositivo não pôde ser reconfigurado (%v).\n"+
					"  Vídeo pode aparecer embaralhado no OBS.\n\n"+
					"  SOLUÇÃO: feche o OBS, pressione 'r' para reiniciar,\n"+
					"  depois reabra o OBS — ele irá negociar %dx%d corretamente.",
				err, outW, outH,
			)
		}

		args := []string{
			"--serial", serial,
			"--video-source=camera",
			"--camera-facing=" + facing,
			"--no-audio",
			"--no-playback",
			"--v4l2-sink=" + videoDevice,
			"--video-codec=h264",
			fmt.Sprintf("--camera-size=%dx%d", cs.w, cs.h),
		}

		if facing == "front" && rotation != "" {
			args = append(args, "--capture-orientation="+rotation)
		}

		cmd := exec.Command("scrcpy", args...)
		if err := cmd.Start(); err != nil {
			return scrcpyStartedMsg{err: fmt.Errorf("falha ao iniciar scrcpy: %w", err)}
		}
		return scrcpyStartedMsg{cmd: cmd, formatWarn: formatWarn}
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro: %v", err)
		os.Exit(1)
	}
}
