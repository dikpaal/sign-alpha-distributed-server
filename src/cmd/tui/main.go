package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const serverURL = "http://localhost:8080"

// Styles
var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2)

	priceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	upStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	downStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))
)

// API response types
type PriceResponse struct {
	Price float64 `json:"price"`
}

type StatsResponse struct {
	MovingAverage float64 `json:"moving_average"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
}

type SymbolResponse struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// Dashboard data
type DashboardData struct {
	Symbol        string
	CoinName      string
	Price         float64
	PrevPrice     float64
	High          float64
	Low           float64
	MovingAverage float64
	Change        float64
	ChangePercent float64
	Connected     bool
	Error         string
}

// Messages
type tickMsg time.Time
type dataMsg DashboardData

// Model
type model struct {
	data     DashboardData
	history  []float64
	quitting bool
}

func initialModel() model {
	return model{
		history: make([]float64, 0, 20),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchData(), tick())
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchData() tea.Cmd {
	return func() tea.Msg {
		data := DashboardData{}

		// Fetch symbol info
		symbolResp, err := http.Get(serverURL + "/api/symbol")
		if err != nil {
			data.Error = "Server not running. Start with 'make run'"
			return dataMsg(data)
		}
		defer symbolResp.Body.Close()

		var symbolData SymbolResponse
		if err := json.NewDecoder(symbolResp.Body).Decode(&symbolData); err == nil {
			data.Symbol = symbolData.Symbol
			data.CoinName = symbolData.Name
		}

		// Fetch price
		priceResp, err := http.Get(serverURL + "/api/price")
		if err != nil {
			data.Error = "Failed to fetch price"
			return dataMsg(data)
		}
		defer priceResp.Body.Close()

		var priceData PriceResponse
		if err := json.NewDecoder(priceResp.Body).Decode(&priceData); err == nil {
			data.Price = priceData.Price
		}

		// Fetch stats
		statsResp, err := http.Get(serverURL + "/api/stats")
		if err != nil {
			data.Error = "Failed to fetch stats"
			return dataMsg(data)
		}
		defer statsResp.Body.Close()

		var statsData StatsResponse
		if err := json.NewDecoder(statsResp.Body).Decode(&statsData); err == nil {
			data.MovingAverage = statsData.MovingAverage
			data.High = statsData.High
			data.Low = statsData.Low
		}

		data.Connected = true
		return dataMsg(data)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		return m, tea.Batch(fetchData(), tick())

	case dataMsg:
		newData := DashboardData(msg)

		// Calculate change
		if m.data.Price > 0 && newData.Price > 0 {
			newData.Change = newData.Price - m.data.Price
			newData.ChangePercent = (newData.Change / m.data.Price) * 100
		}
		newData.PrevPrice = m.data.Price

		m.data = newData

		// Update history
		if newData.Price > 0 {
			m.history = append(m.history, newData.Price)
			if len(m.history) > 20 {
				m.history = m.history[1:]
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Error state
	if m.data.Error != "" {
		content := fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			headerStyle.Render("◆ Trading Pipeline Dashboard"),
			errorStyle.Render(m.data.Error),
			helpStyle.Render("Press 'q' to quit"),
		)
		return boxStyle.Render(content)
	}

	// Waiting for data
	if !m.data.Connected {
		content := fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			headerStyle.Render("◆ Trading Pipeline Dashboard"),
			labelStyle.Render("Connecting to server..."),
			helpStyle.Render("Press 'q' to quit"),
		)
		return boxStyle.Render(content)
	}

	// Header
	coinName := m.data.CoinName
	if coinName == "" {
		coinName = "Crypto"
	}
	header := headerStyle.Render(fmt.Sprintf("◆ %s Real-Time Dashboard", coinName))

	// Price display
	priceStr := fmt.Sprintf("$%.2f", m.data.Price)
	if m.data.Price < 1 {
		priceStr = fmt.Sprintf("$%.6f", m.data.Price)
	}

	// Change indicator
	var changeStr string
	if m.data.Change > 0 {
		changeStr = upStyle.Render(fmt.Sprintf("▲ +%.2f (+%.4f%%)", m.data.Change, m.data.ChangePercent))
	} else if m.data.Change < 0 {
		changeStr = downStyle.Render(fmt.Sprintf("▼ %.2f (%.4f%%)", m.data.Change, m.data.ChangePercent))
	} else {
		changeStr = labelStyle.Render("━ 0.00 (0.00%)")
	}

	priceDisplay := priceStyle.Render(priceStr) + "  " + changeStr

	// Stats
	stats := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("Moving Avg:"),
		valueStyle.Render(fmt.Sprintf("$%.2f", m.data.MovingAverage)),
		labelStyle.Render("Session High:"),
		upStyle.Render(fmt.Sprintf("$%.2f", m.data.High)),
		labelStyle.Render("Session Low:"),
		downStyle.Render(fmt.Sprintf("$%.2f", m.data.Low)),
		labelStyle.Render("Spread:"),
		valueStyle.Render(fmt.Sprintf("$%.2f", m.data.High-m.data.Low)),
	)

	// Sparkline
	sparkline := m.renderSparkline()

	// Combine
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s%s\n\n%s",
		header,
		priceDisplay,
		stats,
		labelStyle.Render("Price History: "),
		sparkline,
		helpStyle.Render("Press 'q' to quit"),
	)

	return boxStyle.Render(content)
}

func (m model) renderSparkline() string {
	if len(m.history) < 2 {
		return labelStyle.Render("waiting for data...")
	}

	min, max := m.history[0], m.history[0]
	for _, v := range m.history {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var spark string
	rang := max - min
	if rang == 0 {
		rang = 1
	}

	for i, v := range m.history {
		normalized := (v - min) / rang
		idx := int(normalized * float64(len(chars)-1))
		if idx >= len(chars) {
			idx = len(chars) - 1
		}

		char := string(chars[idx])
		if i > 0 && v > m.history[i-1] {
			spark += upStyle.Render(char)
		} else if i > 0 && v < m.history[i-1] {
			spark += downStyle.Render(char)
		} else {
			spark += valueStyle.Render(char)
		}
	}

	return spark
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
