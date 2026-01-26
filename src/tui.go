package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Available trading pairs
var coins = []struct {
	symbol string
	name   string
}{
	{"btcusdt", "Bitcoin (BTC)"},
	{"ethusdt", "Ethereum (ETH)"},
	{"solusdt", "Solana (SOL)"},
	{"bnbusdt", "Binance Coin (BNB)"},
	{"xrpusdt", "Ripple (XRP)"},
	{"dogeusdt", "Dogecoin (DOGE)"},
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("10")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)
)

// Model for the TUI
type model struct {
	cursor   int
	selected string
	done     bool
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles input
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(coins)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = coins[m.cursor].symbol
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the TUI
func (m model) View() string {
	s := titleStyle.Render("Select Cryptocurrency to Track") + "\n\n"

	for i, coin := range coins {
		cursor := "  "
		style := itemStyle
		if m.cursor == i {
			cursor = "▸ "
			style = selectedStyle
		}
		s += style.Render(fmt.Sprintf("%s%s", cursor, coin.name)) + "\n"
	}

	s += helpStyle.Render("\n↑/↓: navigate • enter: select • q: quit")
	return s
}

// RunTUI runs the coin selection TUI and returns the selected symbol
func RunTUI() (string, error) {
	m := model{cursor: 0}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(model)
	if result.selected == "" {
		return "", fmt.Errorf("no coin selected")
	}

	return result.selected, nil
}

// GetCoinName returns the display name for a symbol
func GetCoinName(symbol string) string {
	for _, coin := range coins {
		if coin.symbol == symbol {
			return coin.name
		}
	}
	return symbol
}
