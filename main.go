package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MenuItem struct {
	Name        string      `json:"name"`
	Command     string      `json:"command,omitempty"`
	Description string      `json:"description,omitempty"`
	Submenu     []*MenuItem `json:"submenu,omitempty"`
	IsExec      bool        `json:"is_executable"`
}

type OCNavigator struct {
	app             *tview.Application
	mainFlex        *tview.Flex
	mainLayout      *tview.Flex
	menuList        *tview.List
	menuFooter      *tview.TextView
	detailView      *tview.TextView
	commandView     *tview.TextView
	statusBar       *tview.TextView
	currentMenu     []*MenuItem
	menuStack       [][]*MenuItem
	titleStack      []string
	currentContext  string
	currentProject  string
	commandHistory  []string
	outputBuffer    strings.Builder
}

func NewOCNavigator() *OCNavigator {
	nav := &OCNavigator{
		app:            tview.NewApplication(),
		menuStack:      make([][]*MenuItem, 0),
		titleStack:     make([]string, 0),
		commandHistory: make([]string, 0),
	}

	nav.getCurrentContext()
	nav.getCurrentProject()
	nav.initializeUI()
	nav.buildMainMenu()

	return nav
}

func (nav *OCNavigator) getCurrentContext() {
	cmd := exec.Command("oc", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		nav.currentContext = "Unknown"
	} else {
		nav.currentContext = strings.TrimSpace(string(output))
	}
}

func (nav *OCNavigator) getCurrentProject() {
	cmd := exec.Command("oc", "project", "-q")
	output, err := cmd.Output()
	if err != nil {
		nav.currentProject = "default"
	} else {
		nav.currentProject = strings.TrimSpace(string(output))
	}
}

func (nav *OCNavigator) initializeUI() {
	// Create main components
	nav.menuList = tview.NewList().ShowSecondaryText(true)
	nav.menuFooter = tview.NewTextView().SetText("- Kini").SetTextAlign(tview.AlignCenter)
	nav.detailView = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	nav.commandView = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	nav.statusBar = tview.NewTextView().SetDynamicColors(true)

	// Style components
	nav.menuList.SetBorder(true).SetTitle(" Navigation ").SetTitleAlign(tview.AlignLeft)
	nav.menuFooter.SetBorder(true).SetBorderPadding(0, 0, 1, 1)
	nav.detailView.SetBorder(true).SetTitle(" Details ").SetTitleAlign(tview.AlignLeft)
	nav.commandView.SetBorder(true).SetTitle(" Command Output ").SetTitleAlign(tview.AlignLeft)

	// Create left panel with menu and footer
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nav.menuList, 0, 1, true).
		AddItem(nav.menuFooter, 3, 0, false)

	// Create right panel with details and command output
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nav.detailView, 0, 1, false).
		AddItem(nav.commandView, 0, 1, false)

	nav.mainFlex = tview.NewFlex().
		AddItem(leftPanel, 0, 1, true).
		AddItem(rightPanel, 0, 2, false)

	nav.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nav.mainFlex, 0, 1, true).
		AddItem(nav.statusBar, 1, 0, false)

	// Set up event handlers
	nav.menuList.SetSelectedFunc(nav.onMenuSelect)
	nav.menuList.SetChangedFunc(nav.onMenuChange)

	// Global key bindings
	nav.app.SetInputCapture(nav.handleGlobalKeys)

	nav.app.SetRoot(nav.mainLayout, true)
	nav.updateStatusBar()
}

func (nav *OCNavigator) buildMainMenu() {
	nav.currentMenu = []*MenuItem{
		{
			Name:        "Projects & Namespaces",
			Description: "Manage OpenShift projects and namespaces",
			Submenu: []*MenuItem{
				{Name: "List all projects", Command: "oc get projects", Description: "Show all available projects", IsExec: true},
				{Name: "Current project info", Command: "oc project", Description: "Display current project information", IsExec: true},
				{Name: "Switch project", Command: "", Description: "Interactive project switching", IsExec: false},
				{Name: "Create new project", Command: "", Description: "Create a new OpenShift project", IsExec: false},
				{Name: "Delete project", Command: "", Description: "Delete an existing project", IsExec: false},
			},
		},
		{
			Name:        "Workloads",
			Description: "Manage application workloads",
			Submenu: []*MenuItem{
				{Name: "Pods", Command: "oc get pods", Description: "List all pods in current namespace", IsExec: true},
				{Name: "Deployments", Command: "oc get deployments", Description: "List all deployments", IsExec: true},
				{Name: "DeploymentConfigs", Command: "oc get dc", Description: "List all deployment configs", IsExec: true},
				{Name: "ReplicaSets", Command: "oc get rs", Description: "List all replica sets", IsExec: true},
				{Name: "StatefulSets", Command: "oc get sts", Description: "List all stateful sets", IsExec: true},
				{Name: "DaemonSets", Command: "oc get ds", Description: "List all daemon sets", IsExec: true},
				{Name: "Jobs", Command: "oc get jobs", Description: "List all jobs", IsExec: true},
				{Name: "CronJobs", Command: "oc get cronjobs", Description: "List all cron jobs", IsExec: true},
			},
		},
		{
			Name:        "Services & Routes",
			Description: "Manage networking and access",
			Submenu: []*MenuItem{
				{Name: "Services", Command: "oc get svc", Description: "List all services", IsExec: true},
				{Name: "Routes", Command: "oc get routes", Description: "List all routes", IsExec: true},
				{Name: "Ingress", Command: "oc get ingress", Description: "List all ingress resources", IsExec: true},
				{Name: "Endpoints", Command: "oc get endpoints", Description: "List all endpoints", IsExec: true},
				{Name: "NetworkPolicies", Command: "oc get networkpolicies", Description: "List network policies", IsExec: true},
			},
		},
		{
			Name:        "Storage",
			Description: "Manage persistent storage",
			Submenu: []*MenuItem{
				{Name: "Persistent Volumes", Command: "oc get pv", Description: "List all persistent volumes", IsExec: true},
				{Name: "Persistent Volume Claims", Command: "oc get pvc", Description: "List all PVCs", IsExec: true},
				{Name: "Storage Classes", Command: "oc get sc", Description: "List all storage classes", IsExec: true},
				{Name: "Volume Snapshots", Command: "oc get volumesnapshots", Description: "List volume snapshots", IsExec: true},
			},
		},
		{
			Name:        "Configuration",
			Description: "Manage configuration resources",
			Submenu: []*MenuItem{
				{Name: "ConfigMaps", Command: "oc get configmaps", Description: "List all config maps", IsExec: true},
				{Name: "Secrets", Command: "oc get secrets", Description: "List all secrets", IsExec: true},
				{Name: "Service Accounts", Command: "oc get sa", Description: "List all service accounts", IsExec: true},
				{Name: "Role Bindings", Command: "oc get rolebindings", Description: "List role bindings", IsExec: true},
				{Name: "Cluster Role Bindings", Command: "oc get clusterrolebindings", Description: "List cluster role bindings", IsExec: true},
			},
		},
		{
			Name:        "Monitoring & Logs",
			Description: "Monitor applications and view logs",
			Submenu: []*MenuItem{
				{Name: "Events", Command: "oc get events --sort-by=.metadata.creationTimestamp", Description: "Show recent events", IsExec: true},
				{Name: "Node status", Command: "oc get nodes", Description: "Check node status", IsExec: true},
				{Name: "Resource usage", Command: "oc top nodes", Description: "Show resource usage by nodes", IsExec: true},
				{Name: "Pod logs", Command: "", Description: "View pod logs", IsExec: false},
				{Name: "Follow logs", Command: "", Description: "Follow pod logs in real-time", IsExec: false},
			},
		},
		{
			Name:        "Build & Deploy",
			Description: "Manage builds and deployments",
			Submenu: []*MenuItem{
				{Name: "Build Configs", Command: "oc get bc", Description: "List all build configs", IsExec: true},
				{Name: "Builds", Command: "oc get builds", Description: "List all builds", IsExec: true},
				{Name: "Image Streams", Command: "oc get is", Description: "List all image streams", IsExec: true},
				{Name: "Image Stream Tags", Command: "oc get istag", Description: "List image stream tags", IsExec: true},
				{Name: "Templates", Command: "oc get templates", Description: "List all templates", IsExec: true},
			},
		},
		{
			Name:        "Cluster Administration",
			Description: "Cluster-level operations",
			Submenu: []*MenuItem{
				{Name: "Cluster version", Command: "oc get clusterversion", Description: "Show cluster version", IsExec: true},
				{Name: "Cluster operators", Command: "oc get co", Description: "List cluster operators", IsExec: true},
				{Name: "Machine Config Pools", Command: "oc get mcp", Description: "List machine config pools", IsExec: true},
				{Name: "Nodes", Command: "oc get nodes -o wide", Description: "List all nodes with details", IsExec: true},
				{Name: "Namespaces", Command: "oc get namespaces", Description: "List all namespaces", IsExec: true},
			},
		},
		{
			Name:        "Custom Commands",
			Description: "Execute custom oc commands",
			IsExec:      false,
		},
		{
			Name:        "Command History",
			Description: "View previously executed commands",
			IsExec:      false,
		},
	}

	nav.populateMenu()
}

func (nav *OCNavigator) populateMenu() {
	nav.menuList.Clear()
	for _, item := range nav.currentMenu {
		nav.menuList.AddItem(item.Name, item.Description, 0, nil)
	}
}

func (nav *OCNavigator) onMenuSelect(index int, mainText string, secondaryText string, shortcut rune) {
	if index >= len(nav.currentMenu) {
		return
	}

	selectedItem := nav.currentMenu[index]

	if selectedItem.Submenu != nil {
		// Navigate to submenu
		nav.menuStack = append(nav.menuStack, nav.currentMenu)
		nav.titleStack = append(nav.titleStack, nav.menuList.GetTitle())
		nav.currentMenu = selectedItem.Submenu
		nav.populateMenu()
		nav.menuList.SetTitle(fmt.Sprintf(" %s ", selectedItem.Name))
		nav.menuList.SetCurrentItem(0)
	} else if selectedItem.IsExec && selectedItem.Command != "" {
		// Execute command
		nav.executeCommand(selectedItem.Command)
	} else {
		// Handle special cases
		switch selectedItem.Name {
		case "Switch project":
			nav.showProjectSwitchDialog()
		case "Create new project":
			nav.showCreateProjectDialog()
		case "Delete project":
			nav.showDeleteProjectDialog()
		case "Custom Commands":
			nav.showCustomCommandDialog()
		case "Command History":
			nav.showCommandHistory()
		case "Pod logs":
			nav.showPodLogsDialog()
		case "Follow logs":
			nav.showFollowLogsDialog()
		default:
			nav.showItemDetails(selectedItem)
		}
	}
}

func (nav *OCNavigator) onMenuChange(index int, mainText string, secondaryText string, shortcut rune) {
	if index >= len(nav.currentMenu) {
		return
	}
	nav.showItemDetails(nav.currentMenu[index])
}

func (nav *OCNavigator) showItemDetails(item *MenuItem) {
	nav.detailView.Clear()
	fmt.Fprintf(nav.detailView, "[yellow]%s[white]\n\n", item.Name)
	fmt.Fprintf(nav.detailView, "%s\n\n", item.Description)

	if item.Command != "" {
		fmt.Fprintf(nav.detailView, "[cyan]Command:[white] %s\n\n", item.Command)
	}

	if item.Submenu != nil {
		fmt.Fprintf(nav.detailView, "[green]Submenu items:[white]\n")
		for _, subitem := range item.Submenu {
			fmt.Fprintf(nav.detailView, "• %s\n", subitem.Name)
		}
	}

	if item.IsExec {
		fmt.Fprintf(nav.detailView, "\n[green]Press Enter to execute[white]")
	}
}

func (nav *OCNavigator) executeCommand(command string) {
	nav.commandView.Clear()
	nav.setStatus("Executing: " + command)

	// Add to history
	nav.commandHistory = append(nav.commandHistory, command)

	// Show command being executed
	fmt.Fprintf(nav.commandView, "[yellow]$ %s[white]\n\n", command)

	// Execute command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Fprintf(nav.commandView, "[red]Error: %v[white]\n\n", err)
	}

	fmt.Fprintf(nav.commandView, "%s", string(output))
	nav.setStatus("Command completed")
}

func (nav *OCNavigator) showCustomCommandDialog() {
	inputField := tview.NewInputField().
		SetLabel("Enter oc command: ").
		SetFieldWidth(50).
		SetAcceptanceFunc(nil)

	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Execute", func() {
			command := inputField.GetText()
			if command != "" {
				if !strings.HasPrefix(command, "oc ") {
					command = "oc " + command
				}
				nav.executeCommand(command)
			}
			nav.app.SetRoot(nav.mainLayout, true)
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetTitle(" Custom Command ").SetBorder(true)
	nav.app.SetRoot(form, true)
}

func (nav *OCNavigator) showCommandHistory() {
	historyText := "Command History"

	for i, cmd := range nav.commandHistory {
		historyText += fmt.Sprintf("\n%d. %s", i+1, cmd)
	}

	if len(nav.commandHistory) == 0 {
		historyText += "\n\nNo commands in history"
	}

	modal := tview.NewModal().
		SetText(historyText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	nav.app.SetRoot(modal, true)
}

// showProjectSwitchDialog shows an input dialog for switching projects
func (nav *OCNavigator) showProjectSwitchDialog() {
	inputField := tview.NewInputField().
		SetLabel("Enter project name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Switch", func() {
			projectName := inputField.GetText()
			if projectName != "" {
				nav.executeCommand(fmt.Sprintf("oc project %s", projectName))
				nav.getCurrentProject()
				nav.updateStatusBar()
			}
			nav.app.SetRoot(nav.mainLayout, true)
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetBorder(true).SetTitle(" Switch Project ").SetTitleAlign(tview.AlignLeft)
	nav.app.SetRoot(form, true)
}

// showCreateProjectDialog shows an input dialog for creating a new project
func (nav *OCNavigator) showCreateProjectDialog() {
	nameField := tview.NewInputField().
		SetLabel("Project name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	descField := tview.NewInputField().
		SetLabel("Description (optional): ").
		SetFieldWidth(50).
		SetAcceptanceFunc(tview.InputFieldMaxLength(100))

	form := tview.NewForm().
		AddFormItem(nameField).
		AddFormItem(descField).
		AddButton("Create", func() {
			projectName := nameField.GetText()
			description := descField.GetText()
			if projectName != "" {
				cmd := fmt.Sprintf("oc new-project %s", projectName)
				if description != "" {
					cmd += fmt.Sprintf(" --description=\"%s\"", description)
				}
				nav.executeCommand(cmd)
				nav.getCurrentProject()
				nav.updateStatusBar()
			}
			nav.app.SetRoot(nav.mainLayout, true)
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetBorder(true).SetTitle(" Create New Project ").SetTitleAlign(tview.AlignLeft)
	nav.app.SetRoot(form, true)
}

// showDeleteProjectDialog shows an input dialog for deleting a project
func (nav *OCNavigator) showDeleteProjectDialog() {
	inputField := tview.NewInputField().
		SetLabel("Enter project name to delete: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Delete", func() {
			projectName := inputField.GetText()
			if projectName != "" {
				// Show confirmation dialog
				nav.showDeleteConfirmationDialog(projectName)
			} else {
				nav.app.SetRoot(nav.mainLayout, true)
			}
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetBorder(true).SetTitle(" Delete Project ").SetTitleAlign(tview.AlignLeft)
	nav.app.SetRoot(form, true)
}

// showDeleteConfirmationDialog shows a confirmation dialog before deleting a project
func (nav *OCNavigator) showDeleteConfirmationDialog(projectName string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete project '%s'?\nThis action cannot be undone!", projectName)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				nav.executeCommand(fmt.Sprintf("oc delete project %s", projectName))
				nav.getCurrentProject()
				nav.updateStatusBar()
			}
			nav.app.SetRoot(nav.mainLayout, true)
		})

	nav.app.SetRoot(modal, true)
}

// showPodLogsDialog shows an input dialog for viewing pod logs
func (nav *OCNavigator) showPodLogsDialog() {
	inputField := tview.NewInputField().
		SetLabel("Enter pod name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(tview.InputFieldMaxLength(100))

	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("View Logs", func() {
			podName := inputField.GetText()
			if podName != "" {
				nav.executeCommand(fmt.Sprintf("oc logs %s", podName))
			}
			nav.app.SetRoot(nav.mainLayout, true)
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetBorder(true).SetTitle(" View Pod Logs ").SetTitleAlign(tview.AlignLeft)
	nav.app.SetRoot(form, true)
}

// showFollowLogsDialog shows an input dialog for following pod logs
func (nav *OCNavigator) showFollowLogsDialog() {
	inputField := tview.NewInputField().
		SetLabel("Enter pod name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(tview.InputFieldMaxLength(100))

	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Follow Logs", func() {
			podName := inputField.GetText()
			if podName != "" {
				nav.executeCommand(fmt.Sprintf("oc logs -f %s", podName))
			}
			nav.app.SetRoot(nav.mainLayout, true)
		}).
		AddButton("Cancel", func() {
			nav.app.SetRoot(nav.mainLayout, true)
		})

	form.SetBorder(true).SetTitle(" Follow Pod Logs ").SetTitleAlign(tview.AlignLeft)
	nav.app.SetRoot(form, true)
}

func (nav *OCNavigator) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if len(nav.menuStack) > 0 {
			// Go back to previous menu
			nav.currentMenu = nav.menuStack[len(nav.menuStack)-1]
			nav.menuStack = nav.menuStack[:len(nav.menuStack)-1]

			if len(nav.titleStack) > 0 {
				nav.menuList.SetTitle(nav.titleStack[len(nav.titleStack)-1])
				nav.titleStack = nav.titleStack[:len(nav.titleStack)-1]
			} else {
				nav.menuList.SetTitle(" Navigation ")
			}

			nav.populateMenu()
			return nil
		} else {
			nav.app.Stop()
		}
	case tcell.KeyCtrlC:
		nav.app.Stop()
	case tcell.KeyCtrlR:
		nav.getCurrentContext()
		nav.getCurrentProject()
		nav.updateStatusBar()
	case tcell.KeyCtrlH:
		nav.showCommandHistory()
		return nil
	case tcell.KeyCtrlX:
		nav.showCustomCommandDialog()
		return nil
	}
	return event
}

func (nav *OCNavigator) setStatus(message string) {
	nav.updateStatusBar()
	// Flash the message briefly
	go func() {
		originalStatus := nav.statusBar.GetText(false)
		nav.statusBar.SetText(fmt.Sprintf("[yellow]%s[white]", message))
		time.Sleep(2 * time.Second)
		nav.statusBar.SetText(originalStatus)
	}()
}

func (nav *OCNavigator) updateStatusBar() {
	status := fmt.Sprintf(" Context: [cyan]%s[white] | Project: [green]%s[white] | ESC: Back | Ctrl+C: Quit | Ctrl+H: History | Ctrl+X: Custom | Ctrl+R: Refresh ",
		nav.currentContext, nav.currentProject)
	nav.statusBar.SetText(status)
}

func (nav *OCNavigator) Run() error {
	return nav.app.Run()
}

// executeCLICommand runs an external command (like oc) and prints its output to stdout/stderr.
func executeCLICommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Executing: %s %s\n", command, strings.Join(args, " "))
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command '%s %s': %w", command, strings.Join(args, " "), err)
	}
	return nil
}

func main() {
	// Check if oc command is available
	if _, err := exec.LookPath("oc"); err != nil {
		fmt.Println("Error: 'oc' command not found. Please install OpenShift CLI.")
		os.Exit(1)
	}

	switchProjectName := flag.String("project", "", "Switch to the specified OpenShift project before starting UI")
	createProjectName := flag.String("create-project", "", "Create a new OpenShift project with the given name and exit")
	deleteProjectName := flag.String("delete-project", "", "Delete an OpenShift project with the given name and exit")

	flag.Parse()

	if *createProjectName != "" {
		fmt.Printf("Attempting to create project: %s\n", *createProjectName)
		err := executeCLICommand("oc", "new-project", *createProjectName)
		if err != nil {
			log.Fatalf("Error creating project '%s': %v", *createProjectName, err)
		}
		fmt.Printf("Project '%s' creation command executed. Check 'oc projects' to verify.\n", *createProjectName)
		return // Exit after creating
	}

	if *deleteProjectName != "" {
		fmt.Printf("Attempting to delete project: %s\n", *deleteProjectName)
		err := executeCLICommand("oc", "delete", "project", *deleteProjectName)
		if err != nil {
			log.Fatalf("Error deleting project '%s': %v", *deleteProjectName, err)
		}
		fmt.Printf("Project '%s' deletion command executed. Check 'oc projects' to verify.\n", *deleteProjectName)
		return // Exit after deleting
	}

	if *switchProjectName != "" {
		fmt.Printf("Attempting to switch to project: %s\n", *switchProjectName)
		err := executeCLICommand("oc", "project", *switchProjectName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error switching to project '%s': %v. Starting TUI with current project.\n", *switchProjectName, err)
		} else {
			fmt.Printf("Successfully switched to project '%s'.\n", *switchProjectName)
		}
	}

	navigator := NewOCNavigator()
	if err := navigator.Run(); err != nil {
		log.Fatalf("Error running oc-navigator: %v", err)
	}
}
