// Package validate holds logic for the execution of the validate command
package validate

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chelnak/pdk/internal/config"
	"github.com/chelnak/pdk/pkg/tool"
	"github.com/chelnak/pdk/pkg/utils"
	"github.com/chelnak/pdk/pkg/validate/backend/docker"
	"github.com/spf13/afero"
	"golang.org/x/exp/slices"
)

type Validator interface {
	Run() error
	List()
}

type ValidatorOptions struct {
	ToolPath    string
	CodePath    string
	CachePath   string
	ToolArgs    string
	AlwaysBuild bool
	ResultsView string
	Serial      bool
	WorkerCount int
	Group       string
	Args        []string
	List        bool
}

type validator struct {
	ValidatorOptions
	AFS       *afero.Afero
	IOFS      *afero.IOFS
	toolUtils *tool.Utils
}

func NewValidator(options ValidatorOptions) Validator {
	fs := afero.NewOsFs()
	AFS := &afero.Afero{Fs: &afero.Afero{Fs: fs}}
	IOFS := &afero.IOFS{Fs: &afero.Afero{Fs: fs}}
	toolUtils := tool.NewToolUtils(AFS, IOFS)

	return &validator{
		ValidatorOptions: options,
		AFS:              AFS,
		IOFS:             IOFS,
		toolUtils:        toolUtils,
	}
}

type ExitCode int64

// The getToolArgs function returns a map of tool names as keys, and slices of arguments
// for corresponding tools as the values.
func (v *validator) getToolArgs() (map[string][]string, error) {
	toolArgs := make(map[string][]string)

	if v.Group != "" {
		group, err := tool.GetSelectedGroup(path.Join(v.CodePath, "validate.yml"), v.Group)
		if err != nil {
			return toolArgs, err
		}

		for _, t := range group.Tools {
			toolArgs[t.Name] = t.Args
		}
	} else {
		toolArgs[v.Args[0]] = strings.Split(v.ToolArgs, " ")
	}

	return toolArgs, nil
}

// Returns a function that can be in the slices.IndexFunc function for finding the index of a
// config which has been read in.
func (v *validator) findToolIndexFunc(findNamespace tool.Namespace) func(tool.ToolConfig) bool {
	return func(config tool.ToolConfig) bool {
		configNamespace := config.Plugin.Namespace
		return findNamespace.ID == configNamespace.ID &&
			findNamespace.Author == configNamespace.Author &&
			(findNamespace.Version == configNamespace.Version || findNamespace.Version == "")
	}
}

// The findToolConfigs function finds tool configurations for the tools that user has specified.
// TODO This could possibly be made more generic in the future templates can be found in a similar way.
func (v *validator) findToolConfigs(allToolConfigs []tool.ToolConfig, targetToolNamespaces []tool.Namespace) ([]tool.ToolConfig, error) {
	sort.Slice(allToolConfigs, func(i, j int) bool { // Sort configs by version, to pick up latest tool versions by default
		return allToolConfigs[i].Plugin.Version > allToolConfigs[j].Plugin.Version
	})

	var foundTools []tool.ToolConfig
	for _, findNamespace := range targetToolNamespaces {
		foundToolIndex := slices.IndexFunc[tool.ToolConfig](allToolConfigs, v.findToolIndexFunc(findNamespace))

		if foundToolIndex != -1 {
			foundTools = append(foundTools, allToolConfigs[foundToolIndex])
		} else {
			return nil, fmt.Errorf("%s/%s/%s tool cannot be found", findNamespace.Author, findNamespace.ID, findNamespace.Version)
		}
	}

	return foundTools, nil
}

// Parses the namespace of a tool from raw text to the Namespace struct.
// Example of a raw tool name: 'puppetlabs/epp' or 'puppetlabs/epp/0.1.0'.
func (v *validator) getToolNamespaces(rawToolNames []string) ([]tool.Namespace, error) {
	var toolNamespaces []tool.Namespace
	for _, name := range rawToolNames {
		namespace, err := v.toolUtils.GetToolNamespace(name)
		if err != nil {
			return nil, err
		}
		toolNamespaces = append(toolNamespaces, namespace)
	}

	return toolNamespaces, nil
}

// The getTools function takes raw tool names and their arguments and uses the raw tool names
// to import tools using their locally stored configs. An example of a raw tool name would be:
// 'puppetlabs/epp' or 'puppetlabs/epp/0.1.0'.
func (v *validator) getTools() ([]*tool.Tool, error) {
	toolArgs, err := v.getToolArgs()
	if err != nil {
		return nil, err
	}

	rawToolNames := utils.GetMapKeys(toolArgs)
	toolNamespaces, err := v.getToolNamespaces(rawToolNames)
	if err != nil {
		return nil, err
	}

	allToolConfigs := v.toolUtils.ReadToolConfigs(v.ToolPath, true)
	foundToolConfigs, err := v.findToolConfigs(allToolConfigs, toolNamespaces)
	if err != nil {
		return nil, err
	}

	// add arguments to each tool
	var foundTools []*tool.Tool
	for idx, cfg := range foundToolConfigs {
		args := toolArgs[rawToolNames[idx]] // rawToolNames length is the same as foundToolConfigs length, program will error before this if not
		foundTools = append(foundTools, &tool.Tool{Cfg: cfg, Args: args})
	}

	return foundTools, err
}

func (v *validator) validatorFunc(tool *tool.Tool) func() error {
	return func() error {
		backend := &docker.Docker{
			AFS:            v.AFS,
			IOFS:           v.IOFS,
			AlwaysBuild:    v.AlwaysBuild,
			ContextTimeout: time.Duration(config.Config.ToolTimeout),
			CodePath:       v.CodePath,
		}

		return backend.Validate(tool)
	}
}

func (v *validator) createTasks(tools []*tool.Tool) []*Task {
	var tasks []*Task
	for _, t := range tools {
		name := fmt.Sprintf("%s/%s/%s", t.Cfg.Plugin.Author, t.Cfg.Plugin.ID, t.Cfg.Plugin.Version)
		if len(t.Args) > 0 {
			name = fmt.Sprintf("%s, args=%s", name, t.Args)
		}

		tasks = append(tasks, NewTask(name, v.validatorFunc(t)))
	}

	return tasks
}

func cleanOutput(text string) string {
	exp := regexp.MustCompile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
	text = exp.ReplaceAllString(text, "")
	text = strings.TrimPrefix(text, "\n") // Trim prefix newline if it exists
	text = strings.TrimSuffix(text, "\n")
	text = strings.ReplaceAll(text, "/code/", "")
	return text
}

func printOutput(tools []*tool.Tool) {
	for _, t := range tools {
		var errOutput string
		if t.Stderr != "" {
			errOutput = t.Stderr
		} else if t.ExitCode != 0 {
			errOutput = t.Stdout
		}

		if errOutput != "" {
			fmt.Printf("\n%s: %s\n", t.Cfg.Plugin.Display, cleanOutput(errOutput))
		}
	}
}

func (v *validator) Run() error {
	tools, err := v.getTools()
	if err != nil {
		return err
	}

	tasks := v.createTasks(tools)
	pool := NewPool(tasks, v.WorkerCount)
	pool.Run()

	printOutput(tools)

	return nil
}

func (v *validator) List() {
	tableOpts := v.toolUtils.GetTable(v.ToolPath, true)
	utils.RenderTable(tableOpts)
}
