package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type status int

const divisor = 4

const (
	todo status = iota
	inProgress
	done
)

// Model Management
var models []tea.Model

const (
	model status = iota
	form
)

// Styling
var (
	columnStyle = lipgloss.NewStyle().
			Padding(1, 2)
	focusedStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
)

// Task definition
type Task struct {
	status      status
	title       string
	description string
}

func NewTask(status status, title, description string) Task {
	return Task{status: status, title: title, description: description}
}

func (t *Task) Next() {
	if t.status == done {
		t.status = todo
	} else {
		t.status++
	}
}

// Implement list.Item interface
func (t Task) FilterValue() string { return t.title }
func (t Task) Title() string       { return t.title }
func (t Task) Description() string { return t.description }

// Main model
type Model struct {
	loaded   bool
	focused  status
	lists    []list.Model
	quitting bool
}

// New main model
func New() *Model {
	return &Model{}
}

// Move task to the next list
func (m *Model) MoveToNext() tea.Msg {
	selectedItem := m.lists[m.focused].SelectedItem()
	if selectedItem == nil {
		return nil
	}
	selectedTask := selectedItem.(Task)

	// Remove from current list
	m.lists[m.focused].RemoveItem(m.lists[m.focused].Index())

	// Advance status
	selectedTask.Next()

	// Insert into new list at the end
	targetList := m.lists[selectedTask.status]
	targetList.InsertItem(len(targetList.Items()), list.Item(selectedTask))

	// Switch focus to the new list
	m.focused = selectedTask.status

	return nil
}

// Delete selected task
func (m *Model) DeleteTask() tea.Msg {
	if selectedItem := m.lists[m.focused].SelectedItem(); selectedItem != nil {
		m.lists[m.focused].RemoveItem(m.lists[m.focused].Index())
	}
	return nil
}

// Change focus
func (m *Model) Next() {
	if m.focused == done {
		m.focused = todo
	} else {
		m.focused++
	}
}
func (m *Model) Prev() {
	if m.focused == todo {
		m.focused = done
	} else {
		m.focused--
	}
}

// Initialize lists (empty) and disable the default help panel
func (m *Model) initLists(width, height int) {
	listWidth := width / divisor
	listHeight := height / 2 // start smaller, adjust dynamically
	m.lists = []list.Model{
		list.New([]list.Item{}, list.NewDefaultDelegate(), listWidth, listHeight),
		list.New([]list.Item{}, list.NewDefaultDelegate(), listWidth, listHeight),
		list.New([]list.Item{}, list.NewDefaultDelegate(), listWidth, listHeight),
	}

	for i := range m.lists {
		m.lists[i].SetShowHelp(false) // REMOVE the default help/menu
	}

	m.lists[todo].Title = "To Do"
	m.lists[inProgress].Title = "In Progress"
	m.lists[done].Title = "Done"
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			columnStyle.Width(msg.Width / divisor)
			focusedStyle.Width(msg.Width / divisor)
			columnStyle.Height(msg.Height / 2)  // dynamic smaller height
			focusedStyle.Height(msg.Height / 2) // dynamic smaller height
			m.initLists(msg.Width, msg.Height)
			m.loaded = true
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "left", "h":
			m.Prev()
		case "right", "l":
			m.Next()
		case "enter":
			return m, m.MoveToNext
		case "d":
			return m, m.DeleteTask
		case "n":
			models[model] = m // save current
			models[form] = NewForm(m.focused)
			return models[form].Update(nil)
		}
	}

	var cmd tea.Cmd
	m.lists[m.focused], cmd = m.lists[m.focused].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.loaded {
		todoView := m.lists[todo].View()
		inProgView := m.lists[inProgress].View()
		doneView := m.lists[done].View()

		switch m.focused {
		case inProgress:
			return lipgloss.JoinHorizontal(lipgloss.Left,
				columnStyle.Render(todoView),
				focusedStyle.Render(inProgView),
				columnStyle.Render(doneView),
			)
		case done:
			return lipgloss.JoinHorizontal(lipgloss.Left,
				columnStyle.Render(todoView),
				columnStyle.Render(inProgView),
				focusedStyle.Render(doneView),
			)
		default:
			return lipgloss.JoinHorizontal(lipgloss.Left,
				focusedStyle.Render(todoView),
				columnStyle.Render(inProgView),
				columnStyle.Render(doneView),
			)
		}
	}

	return "loading..."
}

// Form model
type Form struct {
	focused     status
	title       textinput.Model
	description textarea.Model
}

func NewForm(focused status) *Form {
	form := &Form{focused: focused}
	form.title = textinput.New()
	form.title.Focus()
	form.description = textarea.New()
	return form
}

func (m Form) CreateTask() tea.Msg {
	return NewTask(m.focused, m.title.Value(), m.description.Value())
}

func (m Form) Init() tea.Cmd { return nil }

func (m Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.title.Focused() {
				m.title.Blur()
				m.description.Focus()
				return m, textarea.Blink
			} else {
				models[form] = m
				return models[model], m.CreateTask
			}
		}
	}

	var cmd tea.Cmd
	if m.title.Focused() {
		m.title, cmd = m.title.Update(msg)
		return m, cmd
	} else {
		m.description, cmd = m.description.Update(msg)
		return m, cmd
	}
}

func (m Form) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.title.View(), m.description.View())
}

func main() {
	models = []tea.Model{New(), NewForm(todo)}
	m := models[model]
	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
