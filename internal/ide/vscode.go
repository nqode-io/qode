package ide

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
)

// SetupVSCode generates VS Code configuration files.
func SetupVSCode(root string, cfg *config.Config) error {
	vscodeDir := filepath.Join(root, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return err
	}

	count := 0

	if cfg.IDE.VSCode.Launch {
		if err := writeJSONFile(filepath.Join(vscodeDir, "launch.json"), buildLaunchJSON(cfg)); err != nil {
			return err
		}
		count++
	}

	if cfg.IDE.VSCode.Tasks {
		if err := writeJSONFile(filepath.Join(vscodeDir, "tasks.json"), buildTasksJSON(cfg)); err != nil {
			return err
		}
		count++
	}

	if cfg.IDE.VSCode.Settings {
		if err := writeJSONFile(filepath.Join(vscodeDir, "settings.json"), buildSettingsJSON(cfg)); err != nil {
			return err
		}
		count++
	}

	if cfg.IDE.VSCode.Extensions {
		if err := writeJSONFile(filepath.Join(vscodeDir, "extensions.json"), buildExtensionsJSON(cfg)); err != nil {
			return err
		}
		count++
	}

	fmt.Printf("  VS Code: .vscode/ (%d files)\n", count)
	return nil
}

func buildLaunchJSON(cfg *config.Config) map[string]interface{} {
	var configs []map[string]interface{}

	for _, layer := range cfg.Layers() {
		switch layer.Stack {
		case "nextjs", "react":
			configs = append(configs, map[string]interface{}{
				"name":              fmt.Sprintf("Debug: %s (%s)", layer.Name, layer.Stack),
				"type":              "node",
				"request":           "launch",
				"runtimeExecutable": "npm",
				"runtimeArgs":       []string{"run", "dev"},
				"cwd":               "${workspaceFolder}/" + layer.Path,
				"env":               map[string]string{"NODE_ENV": "development"},
				"console":           "integratedTerminal",
			})
		case "angular":
			configs = append(configs, map[string]interface{}{
				"name":              fmt.Sprintf("Debug: %s (Angular)", layer.Name),
				"type":              "node",
				"request":           "launch",
				"runtimeExecutable": "ng",
				"runtimeArgs":       []string{"serve"},
				"cwd":               "${workspaceFolder}/" + layer.Path,
				"console":           "integratedTerminal",
			})
		case "dotnet":
			configs = append(configs, map[string]interface{}{
				"name":    fmt.Sprintf("Debug: %s (.NET)", layer.Name),
				"type":    "coreclr",
				"request": "launch",
				"program": "${workspaceFolder}/" + layer.Path + "/bin/Debug/net8.0/${workspaceFolderBasename}.dll",
				"cwd":     "${workspaceFolder}/" + layer.Path,
				"env":     map[string]string{"ASPNETCORE_ENVIRONMENT": "Development"},
			})
		case "java":
			configs = append(configs, map[string]interface{}{
				"name":    fmt.Sprintf("Debug: %s (Java)", layer.Name),
				"type":    "java",
				"request": "launch",
				"cwd":     "${workspaceFolder}/" + layer.Path,
				"console": "integratedTerminal",
			})
		case "python":
			configs = append(configs, map[string]interface{}{
				"name":    fmt.Sprintf("Debug: %s (Python)", layer.Name),
				"type":    "debugpy",
				"request": "launch",
				"program": "${workspaceFolder}/" + layer.Path + "/main.py",
				"cwd":     "${workspaceFolder}/" + layer.Path,
				"env":     map[string]string{"ENV": "development"},
			})
		}
	}

	return map[string]interface{}{
		"version":        "0.2.0",
		"configurations": configs,
	}
}

func buildTasksJSON(cfg *config.Config) map[string]interface{} {
	var tasks []map[string]interface{}

	for _, layer := range cfg.Layers() {
		if layer.Test.Unit != "" {
			tasks = append(tasks, map[string]interface{}{
				"label":   fmt.Sprintf("test: %s", layer.Name),
				"type":    "shell",
				"command": layer.Test.Unit,
				"options": map[string]string{"cwd": "${workspaceFolder}/" + layer.Path},
				"group":   "test",
			})
		}
		if layer.Test.Lint != "" {
			tasks = append(tasks, map[string]interface{}{
				"label":   fmt.Sprintf("lint: %s", layer.Name),
				"type":    "shell",
				"command": layer.Test.Lint,
				"options": map[string]string{"cwd": "${workspaceFolder}/" + layer.Path},
				"group":   "build",
			})
		}
		if layer.Test.Build != "" {
			tasks = append(tasks, map[string]interface{}{
				"label":   fmt.Sprintf("build: %s", layer.Name),
				"type":    "shell",
				"command": layer.Test.Build,
				"options": map[string]string{"cwd": "${workspaceFolder}/" + layer.Path},
				"group":   map[string]interface{}{"kind": "build", "isDefault": true},
			})
		}
	}

	// qode tasks.
	tasks = append(tasks,
		map[string]interface{}{
			"label":   "qode: fetch ticket",
			"type":    "shell",
			"command": "qode ticket fetch ${input:ticketUrl}",
			"group":   "build",
		},
		map[string]interface{}{
			"label":   "qode: check all",
			"type":    "shell",
			"command": "qode check",
			"group":   "test",
		},
		map[string]interface{}{
			"label":   "qode: review code",
			"type":    "shell",
			"command": "qode review code",
			"group":   "test",
		},
		map[string]interface{}{
			"label":   "qode: review security",
			"type":    "shell",
			"command": "qode review security",
			"group":   "test",
		},
	)

	return map[string]interface{}{
		"version": "2.0.0",
		"tasks":   tasks,
		"inputs": []map[string]interface{}{
			{
				"id":          "ticketUrl",
				"type":        "promptString",
				"description": "Ticket URL (GitHub issue, Jira, Linear, Azure DevOps)",
			},
		},
	}
}

func buildSettingsJSON(cfg *config.Config) map[string]interface{} {
	settings := map[string]interface{}{
		"editor.formatOnSave": true,
		"editor.rulers":       []int{120},
	}

	for _, layer := range cfg.Layers() {
		switch layer.Stack {
		case "react", "nextjs":
			settings["typescript.preferences.importModuleSpecifier"] = "relative"
			settings["editor.defaultFormatter"] = "esbenp.prettier-vscode"
		case "angular":
			settings["[typescript]"] = map[string]interface{}{"editor.defaultFormatter": "esbenp.prettier-vscode"}
		case "dotnet":
			settings["[csharp]"] = map[string]interface{}{"editor.defaultFormatter": "ms-dotnettools.csharp"}
		case "java":
			settings["java.format.enabled"] = true
		case "python":
			settings["python.formatting.provider"] = "black"
			settings["[python]"] = map[string]interface{}{"editor.defaultFormatter": "ms-python.black-formatter"}
		}
	}

	return settings
}

func buildExtensionsJSON(cfg *config.Config) map[string]interface{} {
	recs := []string{"editorconfig.editorconfig", "github.copilot"}

	for _, layer := range cfg.Layers() {
		switch layer.Stack {
		case "react", "nextjs":
			recs = append(recs, "dbaeumer.vscode-eslint", "esbenp.prettier-vscode", "bradlc.vscode-tailwindcss")
		case "angular":
			recs = append(recs, "angular.ng-template", "esbenp.prettier-vscode", "dbaeumer.vscode-eslint")
		case "dotnet":
			recs = append(recs, "ms-dotnettools.csdevkit", "ms-dotnettools.csharp")
		case "java":
			recs = append(recs, "vscjava.vscode-java-pack", "vmware.vscode-spring-boot")
		case "python":
			recs = append(recs, "ms-python.python", "ms-python.black-formatter", "ms-python.pylint")
		case "go":
			recs = append(recs, "golang.go")
		}
	}

	return map[string]interface{}{"recommendations": dedup(recs)}
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(data))
}
