// Package docker provides a backend implementation for docker containers
package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chelnak/pdk/internal/config"
	"github.com/chelnak/pdk/pkg/tool"
	"github.com/chelnak/pdk/pkg/validate/backend"

	"github.com/spf13/viper"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type Docker struct {
	// We need to be able to mock the docker client in testing
	Client         DockerClientI
	Context        context.Context
	ContextCancel  func()
	ContextTimeout time.Duration
	AFS            *afero.Afero
	IOFS           *afero.IOFS
	AlwaysBuild    bool
	CodePath       string
}

type DockerClientI interface {
	// All docker client functions must be noted here so they can be mocked
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error)
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ServerVersion(context.Context) (types.Version, error)
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
}

func (d *Docker) GetTool(tool *tool.Tool) error {
	// initialise the docker client
	err := d.initClient()
	if err != nil {
		return err
	}

	// what are we looking for?
	toolImageName := d.ImageName(tool)

	// find out if docker knows about our tool
	list, err := d.Client.ImageList(d.Context, types.ImageListOptions{})

	if err != nil {
		//log.Debug().Msgf("Error listing images: %v", err)
		return err
	}

	foundImage := ""
	for _, image := range list {
		for _, tag := range image.RepoTags {
			if tag == toolImageName {
				//log.Debug().Msgf("Found image: %s", image.ID)
				if !d.AlwaysBuild {
					return nil
				}
				foundImage = image.ID
				break
			}
		}
		if foundImage != "" {
			break
		}
	}

	if d.AlwaysBuild && foundImage != "" {
		log.Info().Msg("Rebuilding image. Please wait...")
		_, err = d.Client.ImageRemove(d.Context, foundImage, types.ImageRemoveOptions{Force: true})
		if err != nil {
			log.Error().Msgf("Error removing docker image: %v", err)
			return err
		}
	}

	// No image found with that configuration
	// we must create it
	fileString, err := d.createDockerfile(tool)
	if err != nil {
		return err
	}
	//log.Debug().Msgf("Creating Dockerfile\n--------------------\n%s--------------------\n", fileString)

	// write the contents of fileString to a Dockerfile stored in the
	// tool path
	filePath := filepath.Join(tool.Cfg.Path, "generated.Dockerfile")
	file, err := d.AFS.Create(filePath)
	if err != nil {
		log.Error().Msgf("Error creating Dockerfile: %v", err)
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error().Msgf("Error closing file: %s", err)
		}
	}()

	// Write fileString contents to filepath
	err = d.AFS.WriteFile(filePath, []byte(fileString), 0644)
	if err != nil {
		log.Error().Msgf("Error copying Dockerfile: %v", err)
		return err
	}

	// create a tar of the tool directory *shrug*
	tar, err := archive.TarWithOptions(tool.Cfg.Path, &archive.TarOptions{})
	if err != nil {
		return err
	}

	// build the image
	imageBuildResponse, err := d.Client.ImageBuild(
		d.Context,
		tar,
		types.ImageBuildOptions{
			Dockerfile: "generated.Dockerfile",
			Tags:       []string{toolImageName},
			Remove:     true,
		})

	if err != nil {
		log.Error().Msgf("Unable to build docker image")
		return err
	}

	defer func() {
		err = imageBuildResponse.Body.Close()
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}()

	// Parse the output from Docker, cleaning up where possible
	scanner := bufio.NewScanner(imageBuildResponse.Body)
	for scanner.Scan() {
		var line map[string]string
		_ = json.Unmarshal(scanner.Bytes(), &line) // nolint:errcheck // we don't care about the error here
		printLine := strings.TrimSuffix(line["stream"], "\n")
		if printLine != "" {
			log.Debug().Msgf("%s", printLine)
		}
	}

	return nil
}

func (d *Docker) createDockerfile(tool *tool.Tool) (string, error) {
	// create a dockerfile from the Tool and d.AFS
	dockerfile := strings.Builder{}
	dockerfile.WriteString(fmt.Sprintf("FROM puppet/puppet-agent:%s\n", config.Config.PuppetVersion))

	rubyVersion, err := getRubyVersion(config.Config.PuppetVersion)
	if err != nil {
		return "", err
	}

	if strings.Split(config.Config.PuppetVersion, ".")[0] == "5" {
		dockerfile.WriteString("RUN apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 4528B6CD9E61EF26\n")
	}

	if tool.Cfg.Common.RequiresGit || (tool.Cfg.Gem != nil && tool.Cfg.Gem.BuildTools) {
		dockerfile.WriteString("RUN apt update\n")
	}

	if tool.Cfg.Common.RequiresGit {
		dockerfile.WriteString("RUN apt install git -y\n")
	}

	if tool.Cfg.Gem != nil {
		if tool.Cfg.Gem.BuildTools {
			dockerfile.WriteString("RUN apt install build-essential -y\n")
		}

		dockerfile.WriteString("RUN /opt/puppetlabs/puppet/bin/gem install bundler --no-document\n")

		for _, gem := range tool.Cfg.Gem.Name {
			// is there a compatibility matrix?
			if len(tool.Cfg.Gem.Compatibility) > 0 {
				// is our current version of ruby in the matrix?
				if val, ok := tool.Cfg.Gem.Compatibility[rubyVersion]; ok {
					// is the gem we want to install listed in the matrix?
					if compat, ok := val[gem]; ok {
						dockerfile.WriteString(fmt.Sprintf("RUN /opt/puppetlabs/puppet/bin/gem install %s -f --conservative --minimal-deps -v '%s' --no-document\n", gem, compat))
						continue
					}
				}
			}
			// just install the latest gem
			dockerfile.WriteString(fmt.Sprintf("RUN /opt/puppetlabs/puppet/bin/gem install %s -f --conservative --minimal-deps --no-document\n", gem))
		}
	}

	for key, val := range tool.Cfg.Common.Env {
		dockerfile.WriteString(fmt.Sprintf("ENV %s=\"%s\"\n", key, val))
	}

	// Copy the tools content into the image
	if _, err := d.AFS.Stat(filepath.Join(tool.Cfg.Path, "/content")); err == nil {
		dockerfile.WriteString("COPY ./content/* /tmp/ \n")
	}

	dockerfile.WriteString("VOLUME [ /code, /cache ]\n")
	dockerfile.WriteString("WORKDIR /code\n")

	if tool.Cfg.Common.UseScript != "" {
		// todo: handle ps1 scripts
		dockerfile.WriteString(fmt.Sprintf("ENTRYPOINT [\"/tmp/%s.sh\"]\n", tool.Cfg.Common.UseScript))
	} else {
		if tool.Cfg.Gem != nil {
			dockerfile.WriteString(fmt.Sprintf("ENTRYPOINT [ \"/opt/puppetlabs/puppet/bin/%s\"]\n", tool.Cfg.Gem.Executable))
		}
	}

	if len(tool.Cfg.Common.DefaultArgs) > 0 {
		dockerfile.WriteString(fmt.Sprintf("CMD [\"%s\"]\n", strings.Join(tool.Cfg.Common.DefaultArgs, "\", \"")))
	}

	return dockerfile.String(), nil
}

// ImageName Creates a unique name for the image based on the tool and the PRM configuration
func (d *Docker) ImageName(tool *tool.Tool) string {
	// build up a name based on the tool and puppet version
	imageName := fmt.Sprintf("pdk:puppet-%s_%s-%s_%s", config.Config.PuppetVersion, tool.Cfg.Plugin.Author, tool.Cfg.Plugin.ID, tool.Cfg.Plugin.Version)
	return imageName
}

func getOutputAsStrings(tool *tool.Tool, reader io.ReadCloser) error {
	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)

	_, err := stdcopy.StdCopy(stdoutBuffer, stderrBuffer, reader)
	if err != nil {
		return err
	}

	tool.Stdout = stdoutBuffer.String()
	tool.Stderr = stderrBuffer.String()
	return nil
}

func (d *Docker) setTimeoutContext() (context.Context, context.CancelFunc) {
	timeout := viper.GetInt("toolTimeout")
	if timeout <= 0 {
		timeout = 1800
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	return ctx, cancel
}

func (d *Docker) Validate(tool *tool.Tool) error {
	err := d.GetTool(tool)
	if err != nil {
		return err
	}

	// is Docker up and running?
	status := d.Status()
	if !status.IsAvailable {
		log.Error().Msgf("Docker is not available")
		return fmt.Errorf("%s", status.StatusMsg)
	}

	// clean up paths
	codeDir, _ := filepath.Abs(d.CodePath)
	//log.Debug().Msgf("Code path: %s", codeDir)
	cacheDir, _ := filepath.Abs(config.Config.CachePath)
	//log.Debug().Msgf("Cache path: %s", cacheDir)

	// stand up a container
	containerConf := container.Config{
		Image: d.ImageName(tool),
		Tty:   false,
	}

	if len(tool.Args) > 0 {
		containerConf.Cmd = tool.Args
	}

	timeoutCtx, cancelFunc := d.setTimeoutContext()
	defer cancelFunc()
	resp, err := d.Client.ContainerCreate(timeoutCtx,
		&containerConf,
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: codeDir,
					Target: "/code",
				},
				{
					Type:   mount.TypeBind,
					Source: cacheDir,
					Target: "/cache",
				},
			},
		}, nil, nil, "")

	if err != nil {
		return err
	}
	// the autoremove functionality is too aggressive
	// it fires before we can get at the logs
	defer func() {
		newContext := context.Background() // allows container to be removed after the tool times out
		duration := time.Duration(0)
		err := d.Client.ContainerStop(newContext, resp.ID, &duration)
		if err != nil {
			log.Error().Msgf("Error stopping container: %s", err)
		}

		err = d.Client.ContainerRemove(newContext, resp.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
		})
		if err != nil {
			log.Error().Msgf("Error removing container: %s", err)
		}
	}()

	if err := d.Client.ContainerStart(timeoutCtx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	isError := make(chan error)
	toolExit := make(chan container.ContainerWaitOKBody)
	go func() {
		statusCh, errCh := d.Client.ContainerWait(timeoutCtx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			isError <- err
		case status := <-statusCh:
			toolExit <- status
		}
	}()

	// parse out the containers logs while we wait for the container to finish
	out, err := d.Client.ContainerLogs(timeoutCtx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: "all", Follow: true})
	if err != nil {
		return err
	}

	for {
		err = getOutputAsStrings(tool, out)
		if err != nil {
			return err
		}

		select {
		case err := <-isError:
			return err
		case exitValues := <-toolExit:
			tool.ExitCode = exitValues.StatusCode
			if exitValues.StatusCode == int64(tool.Cfg.Common.SuccessExitCode) {
				return nil
			} else {
				if tool.Stderr != "" {
					err = fmt.Errorf("%s", tool.Stderr)
				} else {
					err = fmt.Errorf("")
				}
				return err
			}
		}
	}
}

func (d *Docker) Exec(tool *tool.Tool) error {
	err := d.GetTool(tool)
	if err != nil {
		return err
	}

	// is Docker up and running?
	status := d.Status()
	if !status.IsAvailable {
		log.Error().Msgf("Docker is not available")
		return fmt.Errorf("%s", status.StatusMsg)
	}

	// clean up paths
	codeDir, _ := filepath.Abs(d.CodePath)
	log.Info().Msgf("Code path: %s", codeDir)
	cacheDir, _ := filepath.Abs(config.Config.CachePath)
	log.Info().Msgf("Cache path: %s", cacheDir)

	log.Info().Msgf("Additional Args: %v", tool.Args)

	// stand up a container
	containerConf := container.Config{
		Image: d.ImageName(tool),
		Tty:   false,
	}
	// args can override the default CMD
	if len(tool.Args) > 0 {
		containerConf.Cmd = tool.Args
	}

	timeoutCtx, cancelFunc := d.setTimeoutContext()
	defer cancelFunc()
	resp, err := d.Client.ContainerCreate(timeoutCtx,
		&containerConf,
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: codeDir,
					Target: "/code",
				},
				{
					Type:   mount.TypeBind,
					Source: cacheDir,
					Target: "/cache",
				},
			},
		}, nil, nil, "")

	if err != nil {
		return err
	}
	// the autoremove functionality is too aggressive
	// it fires before we can get at the logs
	defer func() {
		newContext := context.Background() // allows container to be removed after the tool times out
		duration := time.Duration(1)
		err := d.Client.ContainerStop(newContext, resp.ID, &duration)
		if err != nil {
			log.Error().Msgf("Error stopping container: %s", err)
		}

		err = d.Client.ContainerRemove(newContext, resp.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
		})
		if err != nil {
			log.Error().Msgf("Error removing container: %s", err)
		}
	}()

	if err := d.Client.ContainerStart(timeoutCtx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	isError := make(chan error)
	toolExit := make(chan container.ContainerWaitOKBody)
	go func() {
		statusCh, errCh := d.Client.ContainerWait(timeoutCtx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			isError <- err
		case status := <-statusCh:
			toolExit <- status
		}
	}()

	// parse out the containers logs while we wait for the container to finish
	for {
		out, err := d.Client.ContainerLogs(timeoutCtx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: "all", Follow: true})
		if err != nil {
			return err
		}

		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		if err != nil {
			return err
		}

		select {
		case err := <-isError:
			return err
		case exitValues := <-toolExit:
			tool.ExitCode = exitValues.StatusCode
			if exitValues.StatusCode == int64(tool.Cfg.Common.SuccessExitCode) {
				return nil
			} else {
				// If we have more details on why the tool failed, use that info
				if exitValues.Error != nil {
					err = fmt.Errorf("%s", exitValues.Error.Message)
				} else {
					// otherwise, just log the exit code
					err = fmt.Errorf("tool exited with code: %d", exitValues.StatusCode)
				}
				return err
			}
		}
	}
}

func (d *Docker) initClient() (err error) {
	if d.Client == nil {
		cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)

		if err != nil {
			return err
		}

		ctx := context.Background()

		d.Client = cli
		d.Context = ctx
		d.ContextCancel = nil
	}
	return nil
}

// Status Check to see if the Docker runtime is available:
// if so, return true and info about Docker on this node;
// if not, return false and the error message
func (d *Docker) Status() backend.Status {
	err := d.initClient()
	if err != nil {
		return backend.Status{
			IsAvailable: false,
			StatusMsg:   fmt.Sprintf("unable to initialize the docker client: %s", err.Error()),
		}
	}
	// The client does not error on creation if the background service is not running,
	// but attempting to list the containers does.
	dockerInfo, err := d.Client.ServerVersion(d.Context)
	if err != nil {
		// message := fmt.Sprintf("%s", err)
		message := err.Error()
		// This is 90% likely the reason this command fails;
		// the alternative error message is lengthy and includes
		// references to pipes and the API which are more likely
		// to confuse than help; so trim it to the most useful info.
		daemonNotRunning := "error during connect: This error may indicate that the docker daemon is not running."
		if strings.Contains(message, daemonNotRunning) {
			message = daemonNotRunning
		}
		return backend.Status{
			IsAvailable: false,
			StatusMsg:   message,
		}
	}
	status := fmt.Sprintf("\tPlatform: %s\n\tVersion: %s\n\tAPI Version: %s", dockerInfo.Platform.Name, dockerInfo.Version, dockerInfo.APIVersion)
	return backend.Status{
		IsAvailable: true,
		StatusMsg:   status,
	}
}

func getRubyVersion(version string) (float32, error) {
	major := strings.Split(version, ".")[0]
	majorInt, err := strconv.Atoi(major)
	if err != nil {
		return 0, err
	}

	switch majorInt {
	case 7:
		return 2.7, nil
	case 6:
		return 2.5, nil
	case 5:
		return 2.4, nil
	default:
		return 2.5, nil
	}
}
