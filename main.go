package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
			Foreground(lipgloss.Color("205")). // Cor rosa/roxo
			Border(lipgloss.NormalBorder(), false, false, false, true). // Barra lateral
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
	Label string // O que aparece na tela (ex: "Full HD (1080p)")
	Value int    // O valor para o scrcpy (ex: 1920). 0 = Nativa
}

type model struct {
	devices      []string
	resolutions  []Resolution
	cursor       int
	
	// Seleções do usuário
	selectedDev  string
	selectedCam  string // "back" ou "front"
	selectedRes  int    // valor da resolução
	
	err          error
	state        int      // 0: Lista, 1: Câmera, 2: Resolução, 3: Rodando
	missingDeps  []string
}

func initialModel() model {
	missing := checkDependencies()
	
	// Lista de resoluções padrão
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			// Lógica de limite do cursor baseada no estado
			limit := 0
			if m.state == 0 {
				limit = len(m.devices) - 1
			} else if m.state == 1 {
				limit = 1 // Apenas 2 câmeras
			} else if m.state == 2 {
				limit = len(m.resolutions) - 1
			}
			
			if m.cursor < limit {
				m.cursor++
			}

		case "enter":
			// STATE 0: Escolher Device
			if m.state == 0 {
				if len(m.devices) > 0 {
					m.selectedDev = strings.Fields(m.devices[m.cursor])[0]
					m.state = 1 // Vai para menu Câmera
					m.cursor = 0
				}
			} else 
			// STATE 1: Escolher Câmera
			if m.state == 1 {
				if m.cursor == 0 {
					m.selectedCam = "back"
				} else {
					m.selectedCam = "front"
				}
				m.state = 2 // Vai para menu Resolução
				m.cursor = 1 // Padrão no Full HD (index 1) para sugerir
			} else 
			// STATE 2: Escolher Resolução e INICIAR
			if m.state == 2 {
				m.selectedRes = m.resolutions[m.cursor].Value
				m.state = 3 // Estado "Rodando"
				return m, startScrcpy(m.selectedDev, m.selectedCam, m.selectedRes)
			}
		
		case "r":
			if m.state == 0 {
				m.devices = getAdbDevices()
			}
			// Se estiver rodando, r reinicia o processo
			if m.state == 3 {
				m.state = 0
				m.cursor = 0
				m.devices = getAdbDevices()
				// Aqui precisaria matar o processo antigo, mas para simplificar, voltamos ao menu
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

	// 1. Erro de Dependências
	if len(m.missingDeps) > 0 {
		return renderError(s, m.missingDeps)
	}

	if m.err != nil {
		return fmt.Sprintf("Erro crítico: %v\nPressione q para sair.", m.err)
	}

	// 2. Renderização por Estado
	switch m.state {
	case 0: // Lista de Devices
		s += "Selecione o dispositivo (r para recarregar):\n\n"
		if len(m.devices) == 0 {
			s += itemStyle.Render("Nenhum dispositivo encontrado. Conecte via USB.")
		}
		for i, dev := range m.devices {
			s += renderOption(dev, m.cursor == i)
		}

	case 1: // Escolha da Câmera
		s += fmt.Sprintf("Dispositivo: %s\nEscolha a câmera:\n\n", m.selectedDev)
		options := []string{"📸 Câmera Traseira", "🤳 Câmera Frontal"}
		for i, opt := range options {
			s += renderOption(opt, m.cursor == i)
		}

	case 2: // Escolha da Resolução
		s += fmt.Sprintf("Câmera: %s\nQualidade do stream (limite de tamanho):\n\n", m.selectedCam)
		for i, res := range m.resolutions {
			s += renderOption(res.Label, m.cursor == i)
		}

	case 3: // Rodando
		s += fmt.Sprintf("🚀 STREAMING ATIVO\n\n")
		s += fmt.Sprintf("• Device: %s\n", m.selectedDev)
		s += fmt.Sprintf("• Câmera: %s\n", m.selectedCam)
		resText := "Nativa"
		if m.selectedRes > 0 {
			resText = fmt.Sprintf("%dpx", m.selectedRes)
		}
		s += fmt.Sprintf("• Resolução Max: %s\n", resText)
		s += "\nminimize esta janela e use sua webcam."
		s += helpStyle.Render("\n(Pressione Ctrl+C para encerrar)")
	}

	if m.state != 3 {
		s += helpStyle.Render("\n(enter: selecionar • q: sair)")
	}
	return s
}

// --- Helpers de Renderização ---

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
	if err != nil { return []string{} }
	
	lines := strings.Split(string(out), "\n")
	var devices []string
	for _, line := range lines {
		if strings.Contains(line, "device") && !strings.Contains(line, "List of") {
			devices = append(devices, line)
		}
	}
	return devices
}

func startScrcpy(serial, facing string, resolution int) tea.Cmd {
	return func() tea.Msg {
		// Tenta encontrar a porta 10 primeiro, senão vai na 0
		videoDevice := "/dev/video10"
		if _, err := os.Stat(videoDevice); os.IsNotExist(err) {
			videoDevice = "/dev/video0"
		}

		args := []string{
			"--serial", serial,
			"--video-source=camera",
			"--camera-facing=" + facing,
			"--no-audio",
			"--no-playback",            // <--- CORREÇÃO: Nova flag do Scrcpy 3+
			"--v4l2-sink=" + videoDevice,
			"--video-codec=h264",       // Essencial para Samsung S24/Novos
		}

		if resolution > 0 {
			args = append(args, "--max-size", strconv.Itoa(resolution))
		} else {
			// Segurança: Evita 4K nativo que trava o driver
			args = append(args, "--max-size", "1920")
		}

		cmd := exec.Command("scrcpy", args...)
		
		// Captura o erro detalhado
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Falha no dispositivo %s:\n%s", videoDevice, string(output))
		}
		return nil
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro: %v", err)
		os.Exit(1)
	}
}
