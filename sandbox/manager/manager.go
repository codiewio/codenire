// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"sandbox/internal"

	"github.com/alitto/pond/v2"
	"github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

const imageTagPrefix = "codenire/"
const codenireConfigName = "config.json"

type CodenireImage struct {
	Name  string            `json:"name"`
	Alias string            `json:"alias"`
	Files map[string]string `json:"files"`
	Tar   any               `json:"tar,omitempty"`

	CompileCmd *string `json:"compileCmd,omitempty"`
	RunCmd     string  `json:"runCmd"`
}

type ImageSetupConfig struct {
	Name        string   `json:"name"`
	Labels      []string `json:"labels"`
	Description string   `json:"description"`
	Tar         *string
}

type BuiltImage struct {
	ImageSetupConfig
	Id string
}

type ContainerManager interface {
	Boot() error
	ImageList(prefix string) []string
	GetContainer(ctx context.Context, id string) (*string, error)
	KillAll()
	KillContainer(cId string) error
	Register(i CodenireImage) error
}

type CodenireManager struct {
	sync.Mutex
	numSysWorkers int

	replicaCnt      int
	imageContainers map[string]chan string
	imgs            []BuiltImage

	dockerClient *client.Client
	killSignal   bool
	devMode      bool

	dockerFilesPath string
}

func NewCodenireManager(dev bool, replicCnt int, dockerFilesPath string) *CodenireManager {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic("fail on create docker client")
	}

	return &CodenireManager{
		devMode:         dev,
		dockerClient:    c,
		imageContainers: make(map[string]chan string),
		numSysWorkers:   runtime.NumCPU(),
		replicaCnt:      replicCnt,
		dockerFilesPath: dockerFilesPath,
	}
}

func (m *CodenireManager) Run() error {
	return nil
}

func (m *CodenireManager) Boot() (err error) {
	images := parseConfigFiles(
		m.dockerFilesPath,
		internal.ListDirectories(m.dockerFilesPath),
	)

	pool := pond.NewPool(m.numSysWorkers)
	for i := 0; i < len(images); i++ {
		i := i
		pool.SubmitErr(func() error {
			buildErr := m.buildImage(images[i], m.dockerFilesPath)
			if buildErr != nil {
				log.Println("Build of image failed", "[image]", images[i], "[err]", buildErr)
			}
			log.Println("Build of image success", "[image]", images[i])
			return buildErr
		})
	}

	pool.StopAndWait()

	// TODO:: чекнуть, что все image поднялись

	m.startWorkers()

	return nil
}

func (m *CodenireManager) ImageList(prefix string) []string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Create Docker client failed: %v", err)
	}

	ctx := context.Background()

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		log.Fatalf("Get image list failed: %v", err)
	}

	var ii []string

	for _, i := range images {
		for _, t := range i.RepoTags {
			if strings.Contains(t, prefix) {
				ii = append(ii, t)
			}
		}
	}

	ii = uniq(ii)

	return ii
}

func (m *CodenireManager) GetContainer(ctx context.Context, id string) (*string, error) {
	select {
	case c := <-m.imageContainers[id]:
		return &c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *CodenireManager) KillAll() {
	m.Lock()
	defer m.Unlock()

	m.killSignal = true

	ctx := context.Background()
	containers, err := m.dockerClient.ContainerList(ctx, dockercontainer.ListOptions{All: true})
	if err != nil {
		log.Fatalf("Get Container List failed: %v", err)
	}

	pool := pond.NewPool(m.numSysWorkers)
	for i := 0; i < len(containers); i++ {
		i := i
		pool.Submit(func() {
			ct := containers[i]

			if strings.HasPrefix(ct.Image, imageTagPrefix) {
				fmt.Printf("Stop container %s (ID: %s)...\n", ct.Names[0], ct.ID)

				timeout := 0
				err := m.dockerClient.ContainerStop(ctx, ct.ID, dockercontainer.StopOptions{
					Timeout: &timeout,
				})
				if err != nil {
					log.Printf("Stop container failed %s: %v\n", ct.ID, err)
					return
				}

				fmt.Printf("Container removed: %s\n", ct.ID)
			}
		})
	}
	pool.StopAndWait()
	log.Println("Killed all images")

	m.killSignal = false
}

func (m *CodenireManager) KillContainer(cId string) (err error) {
	timeout := 0
	err = m.dockerClient.ContainerStop(context.Background(), cId, dockercontainer.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *CodenireManager) buildImage(cfg ImageSetupConfig, root string) error {
	tag := fmt.Sprintf("%s%s", imageTagPrefix, cfg.Name)

	buf, err := internal.DirToTar(filepath.Join(root, cfg.Name))
	if err != nil {
		return err
	}

	buildOptions := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		Tags:           []string{tag},
		Labels:         map[string]string{},
		SuppressOutput: !m.devMode,
		Remove:         true,
	}

	buildResponse, err := m.dockerClient.ImageBuild(context.Background(), &buf, buildOptions)
	if err != nil {
		return fmt.Errorf("error building image: %v", err)
	}
	defer buildResponse.Body.Close()

	imageInfo, _, err := m.dockerClient.ImageInspectWithRaw(context.Background(), tag)
	if err != nil {
		return nil
	}

	if len(imageInfo.RepoTags) < 1 {
		return fmt.Errorf("tags not found for %s", cfg.Name)
	}

	t := imageInfo.RepoTags[0]
	builtImage := BuiltImage{
		ImageSetupConfig: cfg,
		Id:               t,
	}
	m.imgs = append(m.imgs, builtImage)

	return nil
}

func (m *CodenireManager) runSndContainer(img BuiltImage) (string, error) {
	ctx := context.Background()

	containerConfig := &dockercontainer.Config{
		Image: img.Id,
		Cmd:   []string{"tail", "-f", "/dev/null"},
	}

	hostConfig := &dockercontainer.HostConfig{
		AutoRemove: true,
		Tmpfs:      map[string]string{"/tmpfs": "exec"},
	}

	name := stripImageName(img.Id)
	name = fmt.Sprintf("play_run_%s_%s", name, internal.RandHex(8))

	containerResp, err := m.dockerClient.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		name,
	)
	if err != nil {
		return "", fmt.Errorf("create container failed: %w", err)
	}

	err = m.dockerClient.ContainerStart(ctx, containerResp.ID, dockercontainer.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("create container failed: %w", err)
	}

	return containerResp.ID, nil
}

func (m *CodenireManager) Register(i CodenireImage) error {
	//------------------------------------------------------------------
	tmpDir, err := os.MkdirTemp("", "docker_image_"+i.Alias)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = internal.CopyFilesToTmpDir(tmpDir, i.Files)
	if err != nil {
		return err
	}

	buf, err := internal.DirToTar(tmpDir)
	if err != nil {
		return err
	}
	//------------------------------------------------------------------

	sprintf := fmt.Sprintf("%s/%s", imageTagPrefix, i.Alias)
	buildOptions := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		Tags:           []string{sprintf},
		Labels:         map[string]string{},
		SuppressOutput: !m.devMode,
	}

	buildResponse, err := m.dockerClient.ImageBuild(context.Background(), &buf, buildOptions)
	if err != nil {
		log.Fatalf("Error building image: %v", err)
	}
	defer buildResponse.Body.Close()

	if m.devMode {
		_, err = io.Copy(os.Stdout, buildResponse.Body)
		if err != nil {
			log.Fatalf("Error reading build response: %v", err)
		}
	}

	fmt.Sprintf("Image %s built successfully!\n", i.Name)

	return nil
}

func (m *CodenireManager) startWorkers() {
	for _, img := range m.imgs {
		m.imageContainers[img.Name] = make(chan string, m.replicaCnt)

		m.startImageWorkers(img)
	}
}

// TODO:: change workers num logic
func (m *CodenireManager) startImageWorkers(img BuiltImage) {

	for i := 0; i < m.replicaCnt; i++ {
		go func() {
			for {
				if m.killSignal {
					continue
				}

				c, err := m.runSndContainer(img)
				if err != nil {
					log.Printf("error starting container: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}

				m.imageContainers[img.Name] <- c
			}
		}()
	}
}

func stripImageName(imgName string) string {
	res := removeAfterColon(imgName)
	parts := strings.Split(res, "/")
	if len(parts) < 2 {
		return res
	}

	return parts[1]
}

func parseConfigFiles(root string, directories []string) []ImageSetupConfig {
	var res []ImageSetupConfig

	for _, d := range directories {
		dir := filepath.Join(root, d)

		info, err := os.Stat(dir)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			continue
		}

		configPath := filepath.Join(dir, codenireConfigName)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		content, err := ioutil.ReadFile(configPath)
		if err != nil {
			continue
		}

		var config ImageSetupConfig
		if err := json.Unmarshal(content, &config); err != nil {
			continue
		}

		base := filepath.Base(dir)
		if config.Name != base {
			continue
		}

		res = append(res, config)
	}

	return res
}

func removeAfterColon(input string) string {
	if idx := strings.Index(input, ":"); idx != -1 {
		return input[:idx]
	}
	return input // Вернем оригинал, если ":" нет
}

func uniq(slice []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, value := range slice {
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}

	return result
}
