package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type item struct {
	name     string
	selected bool
}

type model struct {
	cursor   int
	items    []item
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	s := "Select interfaces (use space to select, enter to confirm):\n\n"
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		checked := " "
		if item.selected {
			checked = "x"
		}
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, item.name)
	}
	s += "\nPress q to quit.\n"
	return s
}

var serviceCmd = &cobra.Command{
	Use:   "service [service_name]",
	Short: "generate a service",
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.MinimumNArgs(1)),
	Run: func(c *cobra.Command, args []string) {
		log.Printf("Generating service: %s", args[0])

		// 1. create folder in /internal/core/service/<service_name>
		// 2. create service.go file in that folder

		serviceName := args[0]
		dirPath := filepath.Join("internal", "core", "service", serviceName)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}

		filePath := filepath.Join(dirPath, "service.go")
		fileContent := fmt.Sprintf("package %s\n\n// Service logic for %s\n", serviceName, serviceName)
		if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
			log.Fatalf("Failed to create service.go: %v", err)
		}
		log.Printf("Created %s", filePath)

		// read go file in internal/core/port/outbound/registry.go
		// extract all interface names and store in a slice using go/parser and go/ast
		registryFilePath := filepath.Join("internal", "core", "port", "outbound", "registry.go")
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, registryFilePath, nil, 0)
		if err != nil {
			log.Fatalf("Failed to parse registry.go: %v", err)
		}

		var interfaceNames []string
		for _, decl := range node.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				_, ok = typeSpec.Type.(*ast.InterfaceType)
				if ok {
					interfaceNames = append(interfaceNames, typeSpec.Name.Name)
				}
			}
		}

		// Prompt user to select one or more interfaces using Bubble Tea
		if len(interfaceNames) == 0 {
			log.Printf("No interfaces found in %s", registryFilePath)
			return
		}

		items := make([]item, len(interfaceNames))
		for i, name := range interfaceNames {
			items[i] = item{name: name}
		}

		p := tea.NewProgram(model{items: items})
		finalModel, err := p.Run()
		if err != nil {
			log.Fatalf("Bubble Tea error: %v", err)
		}
		selected := []string{}
		for _, it := range finalModel.(model).items {
			if it.selected {
				selected = append(selected, it.name)
			}
		}
		log.Printf("Selected interfaces: %v", selected)

		log.Printf("Interfaces found in %s: %v", registryFilePath, interfaceNames)

	},
}

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate code or files",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("use -h to show available commands")
	},
}
