package validate

import (
	"errors"
	"fmt"
	"github.com/chelnak/pdk/pkg/tool"
	"github.com/chelnak/pdk/pkg/utils"
	"github.com/spf13/afero"
	"golang.org/x/exp/slices"
	"path"
	"sort"
	"strings"
	"time"
)

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
	//PluginNamespaces []tool.PluginNamespace
}

type Validator interface {
	Run() error
	List()
}

type validator struct {
	afs afero.Afero
	ValidatorOptions
}

func NewValidator(options ValidatorOptions) Validator {
	fs := afero.NewOsFs()
	afs := afero.Afero{Fs: fs}

	return &validator{
		afs:              afs,
		ValidatorOptions: options,
	}
}

type ValidateExitCode int64

const (
	VALIDATION_PASS ValidateExitCode = iota
	VALIDATION_FAILED
	VALIDATION_ERROR
)

// The getToolArgs function returns a map of tool names as keys, and slices of arguments
// for the corresponding tool as the values.
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
func findToolIndexFunc(findNamespace tool.PluginNamespace) func(tool.ToolConfig) bool {
	return func(config tool.ToolConfig) bool {
		namespace := config.Plugin.PluginNamespace
		if namespace.Id == findNamespace.Id && namespace.Author == findNamespace.Author && (namespace.Version == findNamespace.Version || findNamespace.Version == "") {
			return true
		}
		return false
	}
}

// The findToolConfigs function finds toolConfigs for the tools that user has specified to validate with.
// TODO This could possibly be made more generic in the future templates can be found in a similar way.
func findToolConfigs(allToolConfigs []tool.ToolConfig, findToolNamespaces []tool.PluginNamespace) ([]tool.ToolConfig, error) {
	var foundTools []tool.ToolConfig

	// Sorting allToolConfigs allows the latest versions of tools to be found first
	// if no tool version has been explicitly specified in the PluginNamespace
	sort.Slice(allToolConfigs, func(i, j int) bool {
		return allToolConfigs[i].Plugin.Id > allToolConfigs[j].Plugin.Id
	})

	for _, findNamespace := range findToolNamespaces {
		foundToolIndex := slices.IndexFunc[tool.ToolConfig](allToolConfigs, findToolIndexFunc(findNamespace))

		if foundToolIndex != -1 {
			foundTools = append(foundTools, allToolConfigs[foundToolIndex])
		} else {
			return nil, fmt.Errorf("%s/%s/%s tool cannot be found", findNamespace.Author, findNamespace.Id, findNamespace.Version)
		}
	}

	return foundTools, nil
}

// Parses the namespace of a tool from raw text to the PluginNamespace struct.
// Example of a raw tool name: 'puppetlabs/epp' or 'puppetlabs/epp/0.1.0'.
func getToolNamespaces(rawToolNames []string) ([]tool.PluginNamespace, error) {
	var toolNamespaces []tool.PluginNamespace
	for _, name := range rawToolNames {
		namespace, err := tool.GetToolNamespace(name)
		if err != nil {
			return nil, err
		}
		toolNamespaces = append(toolNamespaces, namespace)
	}

	return toolNamespaces, nil
}

// The getTools function takes raw tool names and their arguments and uses the raw tool names
// to import tools from their configs stored locally. An example of a raw tool name would be:
// 'puppetlabs/epp' or 'puppetlabs/epp/0.1.0'.
func (v *validator) getTools(toolArgs map[string][]string) ([]*tool.Tool, error) {
	rawToolNames := utils.GetMapKeys(toolArgs)
	toolNamespaces, err := getToolNamespaces(rawToolNames)
	if err != nil {
		return nil, err
	}

	allToolConfigs := tool.ReadAllTools(v.ToolPath, true)

	foundToolConfigs, err := findToolConfigs(allToolConfigs, toolNamespaces)
	if err != nil {
		return nil, err
	}

	var foundTools []*tool.Tool
	// add args to tool
	for idx, config := range foundToolConfigs {
		args := toolArgs[rawToolNames[idx]] // rawToolNames will be in same order as foundToolConfigs array
		foundTools = append(foundTools, &tool.Tool{Cfg: config, Args: args})
	}

	return foundTools, err
}

func taskFunc(t *tool.Tool, sleepTime int) func() error {
	return func() error {
		time.Sleep(time.Duration(sleepTime) * time.Second)
		if sleepTime%2 == 0 {
			return errors.New("sample error")
		}
		return nil
	}
}

func createTasks(tools []*tool.Tool) []*Task {
	var tasks []*Task
	for i, t := range tools {
		tasks = append(tasks, NewTask(fmt.Sprintf("%s/%s/%s %s", t.Cfg.Plugin.Version, t.Cfg.Plugin.Id, t.Cfg.Plugin.Author, t.Args), taskFunc(t, i+1)))
	}

	return tasks
}

func (v *validator) Run() error {
	toolArgs, err := v.getToolArgs()
	if err != nil {
		return err
	}

	tools, err := v.getTools(toolArgs)
	if err != nil {
		return err
	}

	tasks := createTasks(tools)
	pool := NewPool(tasks, v.WorkerCount)
	pool.Run()

	return nil
}

func (v *validator) List() {
	tableOpts := tool.GetTable(v.ToolPath, true)
	utils.RenderTable(tableOpts)
}
