package tool

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Group struct {
	ID    string     `yaml:"id"`
	Tools []ToolInfo `yaml:"tools"`
}

type ToolInfo struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type validateYmlContent struct {
	Groups []Group `yaml:"groups"`
}

// getGroupsFromFile gets all groups from a specified validate file.
func getGroupsFromFile(validateFile string) ([]Group, error) {
	fs := afero.NewOsFs()
	afs := afero.Afero{Fs: &afero.Afero{Fs: fs}}

	contentBytes, err := afs.ReadFile(validateFile)
	if err != nil {
		log.Error().Msgf("Error reading validate.yml: %s", err)
		return []Group{}, err
	}

	var contentStruct validateYmlContent
	err = yaml.Unmarshal(contentBytes, &contentStruct)
	if err != nil {
		log.Error().Msgf("validate.yml is not formatted correctly: %s", err)
		return []Group{}, err
	}

	return contentStruct.Groups, nil
}

// GetSelectedGroup gets a group that has been specified by a user from a "validate.yml" file.
func GetSelectedGroup(validateFile string, selectedGroup string) (Group, error) {
	groups, err := getGroupsFromFile(validateFile)
	if err != nil {
		return Group{}, err
	}

	for _, group := range groups {
		if group.ID == selectedGroup {
			return group, nil
		}
	}

	return Group{}, fmt.Errorf("the selected group '%s' cound not be found", selectedGroup)
}
