package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type Mode string

const (
	ModeBrowse Mode = "browse"
	ModeReview Mode = "review"
	ModeAdd    Mode = "add"
	ModeSearch Mode = "search"
	ModeStats  Mode = "stats"
)

type decksMsg struct {
	decks []srs.Deck
	err   error
}

type dueCardsMsg struct {
	cards []srs.Card
	err   error
}

type statsMsg struct {
	stats backend.Stats
	err   error
}

type addCardMsg struct {
	card srs.Card
	err  error
}

type answerMsg struct {
	err error
}

type searchMsg struct {
	cards []srs.Card
	err   error
}

type ioResultMsg struct {
	summary string
	err     error
}

type Model struct {
	backend backend.Backend
	mode    Mode

	keys keyMap
	help help.Model

	theme Theme

	width  int
	height int

	decks      []srs.Deck
	deckCursor int

	reviewDeck  string
	reviewCards []srs.Card
	reviewIdx   int
	revealed    bool

	addInputs    []textinput.Model
	focusedInput int

	searchInput     textinput.Model
	searchResults   []srs.Card
	searchCursor    int
	searchListFocus bool
	searchExecuted  bool

	stats backend.Stats

	helpVisible bool
	errText     string
	infoText    string
}

func NewModel(be backend.Backend) Model {
	m := Model{
		backend: be,
		mode:    ModeBrowse,
		keys:    defaultKeys(),
		help:    help.New(),
		theme:   DefaultTheme(),
	}
	m.help.ShowAll = false
	m.addInputs = buildAddInputs()
	m.searchInput = buildSearchInput()
	return m
}

func buildAddInputs() []textinput.Model {
	placeholders := []string{"Starter German", "Guten Morgen", "Good morning", "greeting,basics", "import yaml ./decks/starter.yaml"}
	inputs := make([]textinput.Model, 0, len(placeholders))
	for i := range placeholders {
		in := textinput.New()
		in.Placeholder = placeholders[i]
		in.CharLimit = 400
		in.Width = 48
		inputs = append(inputs, in)
	}
	inputs[0].Focus()
	return inputs
}

func buildSearchInput() textinput.Model {
	in := textinput.New()
	in.Placeholder = "Type a query and press enter (e.g., hund, greeting)"
	in.CharLimit = 200
	in.Width = 48
	in.Focus()
	return in
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.cmdLoadDecks(), m.cmdLoadStats())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case decksMsg:
		if msg.err != nil {
			m.setError("load decks", msg.err)
			return m, nil
		}
		m.errText = ""
		m.decks = msg.decks
		if m.deckCursor >= len(m.decks) && len(m.decks) > 0 {
			m.deckCursor = len(m.decks) - 1
		}
		return m, nil
	case dueCardsMsg:
		if msg.err != nil {
			m.setError("load due cards", msg.err)
			return m, nil
		}
		m.errText = ""
		m.reviewCards = msg.cards
		m.reviewIdx = 0
		m.revealed = false
		m.mode = ModeReview
		return m, nil
	case statsMsg:
		if msg.err != nil {
			m.setError("load stats", msg.err)
			return m, nil
		}
		m.errText = ""
		m.stats = msg.stats
		return m, nil
	case addCardMsg:
		if msg.err != nil {
			m.setError("add card", msg.err)
			return m, nil
		}
		m.errText = ""
		m.infoText = "Card added to deck: " + msg.card.Deck
		for i := range m.addInputs {
			m.addInputs[i].SetValue("")
		}
		m.setFocusedInput(0)
		return m, tea.Batch(m.cmdLoadDecks(), m.cmdLoadStats())
	case answerMsg:
		if msg.err != nil {
			m.setError("answer card", msg.err)
			return m, nil
		}
		m.errText = ""
		m.reviewIdx++
		m.revealed = false
		return m, tea.Batch(m.cmdLoadDecks(), m.cmdLoadStats())
	case searchMsg:
		if msg.err != nil {
			m.setError("search cards", msg.err)
			m.searchResults = nil
			m.searchCursor = 0
			m.searchListFocus = false
			m.searchExecuted = true
			return m, nil
		}
		m.errText = ""
		m.searchResults = msg.cards
		m.searchCursor = 0
		m.searchExecuted = true
		m.searchListFocus = len(msg.cards) > 0
		return m, nil
	case ioResultMsg:
		if msg.err != nil {
			m.setError("local io", msg.err)
			return m, nil
		}
		m.errText = ""
		m.infoText = msg.summary
		m.addInputs[4].SetValue("")
		return m, tea.Batch(m.cmdLoadDecks(), m.cmdLoadStats())
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.HardQuit) {
			return m, tea.Quit
		}

		// Text entry gets first priority (except hard quit), so printable keys are not intercepted.
		if m.isTextEntryFocused() {
			switch m.mode {
			case ModeAdd:
				return m.updateAdd(msg)
			case ModeSearch:
				return m.updateSearch(msg)
			}
		}

		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		if key.Matches(msg, m.keys.Help) {
			m.helpVisible = !m.helpVisible
			return m, nil
		}
		if key.Matches(msg, m.keys.Back) {
			switch m.mode {
			case ModeReview:
				m.mode = ModeBrowse
			case ModeAdd, ModeSearch, ModeStats:
				m.mode = ModeBrowse
			}
			m.errText = ""
			m.infoText = ""
			return m, nil
		}
		if key.Matches(msg, m.keys.Tab) && m.mode != ModeAdd && m.mode != ModeSearch {
			m.cycleMode()
			if m.mode == ModeStats {
				return m, m.cmdLoadStats()
			}
			return m, nil
		}

		switch m.mode {
		case ModeBrowse:
			return m.updateBrowse(msg)
		case ModeReview:
			return m.updateReview(msg)
		case ModeAdd:
			return m.updateAdd(msg)
		case ModeSearch:
			return m.updateSearch(msg)
		case ModeStats:
			return m.updateStats(msg)
		}
	}
	return m, nil
}

func (m *Model) cycleMode() {
	switch m.mode {
	case ModeBrowse:
		m.mode = ModeAdd
	case ModeAdd:
		m.mode = ModeSearch
	case ModeSearch:
		m.mode = ModeStats
	case ModeStats, ModeReview:
		m.mode = ModeBrowse
	default:
		m.mode = ModeBrowse
	}
}

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.deckCursor < len(m.decks)-1 {
			m.deckCursor++
		}
	case key.Matches(msg, m.keys.Up):
		if m.deckCursor > 0 {
			m.deckCursor--
		}
	case key.Matches(msg, m.keys.Select):
		if len(m.decks) == 0 {
			return m, nil
		}
		m.reviewDeck = m.decks[m.deckCursor].Name
		return m, m.cmdLoadDueCards(m.reviewDeck)
	}
	return m, nil
}

func (m Model) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.reviewCards) == 0 || m.reviewIdx >= len(m.reviewCards) {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Reveal):
		m.revealed = true
		return m, nil
	case key.Matches(msg, m.keys.Again):
		return m.gradeCurrentCard(srs.GradeAgain)
	case key.Matches(msg, m.keys.Hard):
		return m.gradeCurrentCard(srs.GradeHard)
	case key.Matches(msg, m.keys.Good):
		return m.gradeCurrentCard(srs.GradeGood)
	case key.Matches(msg, m.keys.Easy):
		return m.gradeCurrentCard(srs.GradeEasy)
	}
	return m, nil
}

func (m Model) gradeCurrentCard(grade srs.ReviewGrade) (tea.Model, tea.Cmd) {
	if !m.revealed {
		return m, nil
	}
	cardID := m.reviewCards[m.reviewIdx].ID
	return m, m.cmdAnswerCard(cardID, grade)
}

func (m Model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = ModeBrowse
		m.errText = ""
		m.infoText = ""
		return m, nil
	case key.Matches(msg, m.keys.NextInput):
		m.setFocusedInput((m.focusedInput + 1) % len(m.addInputs))
		return m, nil
	case key.Matches(msg, m.keys.PrevInput):
		m.setFocusedInput((m.focusedInput - 1 + len(m.addInputs)) % len(m.addInputs))
		return m, nil
	case key.Matches(msg, m.keys.SubmitCard):
		if m.focusedInput < 3 {
			m.setFocusedInput(m.focusedInput + 1)
			m.infoText = "Moved to next field. Press enter on Tags to submit."
			return m, nil
		}
		if m.focusedInput == 4 {
			raw := strings.TrimSpace(m.addInputs[4].Value())
			cmd, err := parseIOCommand(raw)
			if err != nil {
				m.setError("local io", err)
				return m, nil
			}
			return m, m.cmdRunLocalIO(cmd)
		}
		deck := strings.TrimSpace(m.addInputs[0].Value())
		front := strings.TrimSpace(m.addInputs[1].Value())
		back := strings.TrimSpace(m.addInputs[2].Value())
		tagsRaw := strings.TrimSpace(m.addInputs[3].Value())
		if deck == "" || front == "" || back == "" {
			m.errText = "deck, front, and back are required"
			m.infoText = ""
			return m, nil
		}
		var tags []string
		if tagsRaw != "" {
			for _, t := range strings.Split(tagsRaw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
		return m, m.cmdAddCard(backend.AddCardInput{
			Deck:  deck,
			Front: front,
			Back:  back,
			Tags:  tags,
		})
	}
	m.infoText = ""
	var cmd tea.Cmd
	m.addInputs[m.focusedInput], cmd = m.addInputs[m.focusedInput].Update(msg)
	return m, cmd
}

func (m Model) updateStats(tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = ModeBrowse
		m.errText = ""
		m.infoText = ""
		return m, nil
	case key.Matches(msg, m.keys.NextInput):
		m.searchListFocus = len(m.searchResults) > 0
		return m, nil
	case key.Matches(msg, m.keys.PrevInput):
		m.searchListFocus = false
		return m, nil
	case key.Matches(msg, m.keys.Select):
		query := strings.TrimSpace(m.searchInput.Value())
		if query == "" {
			m.searchExecuted = true
			m.searchResults = nil
			m.searchCursor = 0
			m.searchListFocus = false
			m.errText = ""
			return m, nil
		}
		m.searchListFocus = true
		return m, m.cmdSearchCards(query)
	case key.Matches(msg, m.keys.Down):
		if m.searchListFocus && len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
			return m, nil
		}
	case key.Matches(msg, m.keys.Up):
		if m.searchListFocus && len(m.searchResults) > 0 && m.searchCursor > 0 {
			m.searchCursor--
			return m, nil
		}
	}

	// Any text editing action returns focus to the query input.
	m.searchListFocus = false
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	leftW, rightW := m.paneWidths()
	leftBody := m.viewBrowse()
	if m.mode == ModeSearch {
		leftBody = m.viewSearchList()
	}
	left := m.theme.LeftPane.Width(leftW).Height(maxInt(10, m.height-3)).Render(leftBody)
	var rightBody string
	switch m.mode {
	case ModeReview:
		rightBody = m.viewReview()
	case ModeAdd:
		rightBody = m.viewAdd()
	case ModeSearch:
		rightBody = m.viewSearchDetail()
	case ModeStats:
		rightBody = m.viewStats()
	default:
		rightBody = m.theme.Muted.Render("Select a deck and press enter to start review.")
	}
	right := m.theme.RightPane.Width(rightW).Height(maxInt(10, m.height-3)).Render(rightBody)
	status := m.theme.StatusLine.Render(fmt.Sprintf("mode:%s  time:%s", m.mode, time.Now().UTC().Format("15:04")))
	base := m.theme.App.Render(joinHorizontal(left, right) + "\n" + status + "\n" + m.help.View(m.keys))
	if m.helpVisible {
		return m.renderHelpOverlay(base)
	}
	return base
}

func (m *Model) setFocusedInput(idx int) {
	m.focusedInput = idx
	for i := range m.addInputs {
		if i == m.focusedInput {
			m.addInputs[i].Focus()
		} else {
			m.addInputs[i].Blur()
		}
	}
}

func (m Model) isTextEntryFocused() bool {
	switch m.mode {
	case ModeAdd:
		return true
	case ModeSearch:
		return !m.searchListFocus
	default:
		return false
	}
}

func (m *Model) setError(action string, err error) {
	m.errText = action + ": " + err.Error()
	m.infoText = ""
}

func (m Model) cmdLoadDecks() tea.Cmd {
	return func() tea.Msg {
		decks, err := m.backend.ListDecks(context.Background())
		return decksMsg{decks: decks, err: err}
	}
}

func (m Model) cmdLoadDueCards(deck string) tea.Cmd {
	return func() tea.Msg {
		cards, err := m.backend.DueCards(context.Background(), deck, 50)
		return dueCardsMsg{cards: cards, err: err}
	}
}

func (m Model) cmdLoadStats() tea.Cmd {
	return func() tea.Msg {
		stats, err := m.backend.Stats(context.Background())
		return statsMsg{stats: stats, err: err}
	}
}

func (m Model) cmdAddCard(in backend.AddCardInput) tea.Cmd {
	return func() tea.Msg {
		card, err := m.backend.AddCard(context.Background(), in)
		return addCardMsg{card: card, err: err}
	}
}

func (m Model) cmdAnswerCard(cardID string, grade srs.ReviewGrade) tea.Cmd {
	return func() tea.Msg {
		err := m.backend.AnswerCard(context.Background(), cardID, grade)
		return answerMsg{err: err}
	}
}

func (m Model) cmdSearchCards(query string) tea.Cmd {
	return func() tea.Msg {
		cards, err := m.backend.SearchCards(context.Background(), query, 100)
		return searchMsg{cards: cards, err: err}
	}
}

type localIO interface {
	ImportYAML(ctx context.Context, path string) (int, error)
	ImportCSV(ctx context.Context, path string) (int, error)
	ExportCSV(ctx context.Context, path string) (int, error)
}

func (m Model) cmdRunLocalIO(cmd ioCommand) tea.Cmd {
	return func() tea.Msg {
		be, ok := m.backend.(localIO)
		if !ok {
			return ioResultMsg{err: fmt.Errorf("import/export is available in local backend only")}
		}
		switch cmd.Action {
		case "import":
			switch cmd.Format {
			case "yaml":
				n, err := be.ImportYAML(context.Background(), cmd.Path)
				if err != nil {
					return ioResultMsg{err: err}
				}
				return ioResultMsg{summary: fmt.Sprintf("Imported %d cards from YAML: %s", n, cmd.Path)}
			case "csv":
				n, err := be.ImportCSV(context.Background(), cmd.Path)
				if err != nil {
					return ioResultMsg{err: err}
				}
				return ioResultMsg{summary: fmt.Sprintf("Imported %d cards from CSV: %s", n, cmd.Path)}
			}
		case "export":
			n, err := be.ExportCSV(context.Background(), cmd.Path)
			if err != nil {
				return ioResultMsg{err: err}
			}
			return ioResultMsg{summary: fmt.Sprintf("Exported %d cards to CSV: %s", n, cmd.Path)}
		}
		return ioResultMsg{err: fmt.Errorf("unsupported command")}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
