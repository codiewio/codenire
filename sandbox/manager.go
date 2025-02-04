// Package manager
// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	contract "sandbox/api/gen"
	"sandbox/internal"
)

const imageTagPrefix = "codenire_play/"
const codenireConfigName = "config.json"
const defaultMemoryLimit = 100 << 20

type BuiltImage struct {
	contract.ImageConfig
	Id string
}

type StartedContainer struct {
	CId    string
	Image  BuiltImage
	TmpDir string
}

type NetworkOptions struct {
	Network   string
	ProxyHost string
}

type ContainerManager interface {
	Boot() error
	ImageList() []BuiltImage
	GetContainer(ctx context.Context, id string) (*StartedContainer, error)
	KillAll()
	KillContainer(cId StartedContainer) error
}

type CodenireManager struct {
	sync.Mutex
	numSysWorkers int

	idleContainersCount int
	imageContainers     map[string]chan StartedContainer
	imgs                []BuiltImage

	dockerClient *client.Client
	killSignal   bool
	isolated     bool

	dockerFilesPath string
}

func NewCodenireManager() *CodenireManager {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic("fail on create docker client")
	}

	return &CodenireManager{
		dockerClient:        c,
		imageContainers:     make(map[string]chan StartedContainer),
		numSysWorkers:       runtime.NumCPU(),
		idleContainersCount: *replicaContainerCnt,
		dockerFilesPath:     *dockerFilesPath,
		isolated:            *isolated,
	}
}

func (m *CodenireManager) Run() error {
	return nil
}

func (m *CodenireManager) Boot() (err error) {
	configs := parseConfigFiles(
		m.dockerFilesPath,
		internal.ListDirectories(m.dockerFilesPath),
	)

	pool := pond.NewPool(m.numSysWorkers)
	for i := 0; i < len(configs); i++ {
		i := i
		pool.Submit(func() {

			buildErr := m.buildImage(configs[i], m.dockerFilesPath)
			if buildErr != nil {
				log.Println("Build of Image failed", "[Image]", configs[i].Template, "[err]", buildErr)
				return
			}
		})
	}

	pool.StopAndWait()

	// TODO:: чекнуть, что все Image поднялись

	m.startContainers()

	return nil
}

func (m *CodenireManager) ImageList() []BuiltImage {
	return m.imgs
}

func (m *CodenireManager) GetContainer(ctx context.Context, id string) (*StartedContainer, error) {
	select {
	case c := <-m.getContainer(id):
		return &c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *CodenireManager) KillAll() {
	m.Lock()
	defer m.Unlock()

	m.killSignal = true

	defer func() {
		// TODO:: удалить tmp папки
		m.imageContainers = make(map[string]chan StartedContainer)
		m.killSignal = false
	}()

	ctx := context.Background()
	containers, err := m.dockerClient.ContainerList(ctx, docker.ListOptions{All: true})
	if err != nil {
		log.Fatalf("Get Container List failed: %v", err)
	}

	pool := pond.NewPool(m.numSysWorkers)
	for i := 0; i < len(containers); i++ {
		i := i
		pool.Submit(func() {
			ct := containers[i]

			if !strings.HasPrefix(ct.Image, imageTagPrefix) {
				return
			}

			fmt.Printf("Stop container %s (ID: %s)...\n", ct.Names[0], ct.ID)

			timeout := 0
			err := m.dockerClient.ContainerStop(ctx, ct.ID, docker.StopOptions{
				Timeout: &timeout,
			})
			if err != nil {
				log.Printf("Stop container failed %s: %v\n", ct.ID, err)
				return
			}

			fmt.Printf("Container removed: %s\n", ct.ID)
		})
	}
	pool.StopAndWait()
	log.Println("Killed all images")
}

func (m *CodenireManager) KillContainer(c StartedContainer) (err error) {
	timeout := 0
	err = m.dockerClient.ContainerStop(context.Background(), c.CId, docker.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *CodenireManager) buildImage(cfg contract.ImageConfig, root string) error {
	tag := fmt.Sprintf("%s%s", imageTagPrefix, cfg.Template)

	buf, err := internal.DirToTar(filepath.Join(root, cfg.Template))
	if err != nil {
		return err
	}

	buildOptions := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		Tags:           []string{tag},
		Labels:         map[string]string{},
		SuppressOutput: !*dev,
	}

	buildResponse, err := m.dockerClient.ImageBuild(context.Background(), &buf, buildOptions)
	if err != nil {
		return fmt.Errorf("error building Image: %v", err)
	}
	defer buildResponse.Body.Close()

	scanner := bufio.NewScanner(buildResponse.Body)
	for scanner.Scan() {
		if *dev {
			fmt.Println("[DEBUG BUILD]", scanner.Text())
		}
	}

	imageInfo, _, err := m.dockerClient.ImageInspectWithRaw(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("error on get image info: %v", err)
	}

	if len(imageInfo.RepoTags) < 1 {
		return fmt.Errorf("tags not found for %s", cfg.Template)
	}

	wd := imageInfo.Config.WorkingDir
	if wd == "/" || wd == "" {
		wd = "/app_tmp"
	}
	if cfg.Workdir == "" {
		cfg.Workdir = wd
	}

	builtImage := BuiltImage{
		ImageConfig: cfg,
		Id:          imageInfo.RepoTags[0],
	}
	m.imgs = append(m.imgs, builtImage)

	return nil
}

func (m *CodenireManager) runSndContainer(img BuiltImage) (cont *StartedContainer, err error) {
	ctx := context.Background()

	hostConfig := &docker.HostConfig{
		Runtime:     m.runtime(),
		AutoRemove:  true,
		NetworkMode: docker.NetworkMode(*isolatedNetwork),
		Resources: docker.Resources{
			Memory:     int64(*img.ContainerOptions.MemoryLimit),
			MemorySwap: 0,
		},
	}

	name := stripImageName(img.Id)
	name = fmt.Sprintf("play_run_%s_%s", name, internal.RandHex(8))

	containerConfig := &docker.Config{
		Image: img.Id,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		Env: []string{
			fmt.Sprintf("HTTP_PROXY=%s", *isolatedGateway),
			fmt.Sprintf("HTTPS_PROXY=%s", *isolatedGateway),
		},
	}

	containerResp, err := m.dockerClient.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("create container failed: %w", err)
	}

	err = m.dockerClient.ContainerStart(ctx, containerResp.ID, docker.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("create container failed: %w", err)
	}

	return &StartedContainer{
		CId:   containerResp.ID,
		Image: img,
	}, nil
}

func (m *CodenireManager) startContainers() {
	var ii []string
	for _, img := range m.imgs {
		ii = append(ii, img.Template)
	}
	log.Printf("To start: %s", strings.Join(ii, ","))

	for _, img := range m.imgs {
		for i := 0; i < m.idleContainersCount; i++ {
			go func() {
				for {
					if m.killSignal {
						continue
					}

					c, err := m.runSndContainer(img)
					if err != nil {
						time.Sleep(10 * time.Second)
						continue
					}

					m.getContainer(img.Template) <- *c
				}
			}()
		}
	}

	var cc []string
	for c := range m.imageContainers {
		cc = append(cc, c)
	}
	log.Printf("Run images %s", strings.Join(cc, ","))
}

func (m *CodenireManager) getContainer(template string) chan StartedContainer {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.imageContainers[template]; !exists {
		m.imageContainers[template] = make(chan StartedContainer)
	}
	return m.imageContainers[template]
}

func (m *CodenireManager) runtime() string {
	if m.isolated {
		return "runsc"
	}

	return ""
}

func stripImageName(imgName string) string {
	res := removeAfterColon(imgName)
	parts := strings.Split(res, "/")
	if len(parts) < 2 {
		return res
	}

	return parts[1]
}

func parseConfigFiles(root string, directories []string) []contract.ImageConfig {
	var res []contract.ImageConfig

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
			log.Printf("Parse config err 1: %s", err.Error())
			continue
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("Parse config err 2: %s", err.Error())
			continue
		}

		var config contract.ImageConfig
		if err := json.Unmarshal(content, &config); err != nil {
			log.Printf("Parse config err 3: %s", err.Error())
			continue
		}

		if len(config.Actions) < 1 {
			log.Printf("There are not actions in %s: %s", config.Template, err.Error())
			continue
		}

		config.Provider = "built-in"

		if config.Version == "" {
			config.Version = "1.0"
		}

		memoryLimit := defaultMemoryLimit
		if config.ContainerOptions.MemoryLimit == nil {
			config.ContainerOptions.MemoryLimit = &memoryLimit
		}

		{
			_, defaultExists := config.Actions["default"]
			var first *contract.ImageActionConfig

			for _, actionConfig := range config.Actions {
				if first == nil {
					first = &actionConfig
				}

				if actionConfig.IsDefault && !defaultExists {
					defaultExists = true
					actionConfig.IsDefault = true
					config.Actions["default"] = actionConfig
					continue
				}
			}

			if first != nil && !defaultExists && first.Name != "" {
				config.Actions["default"] = *first
				defaultExists = true
			}

			if !defaultExists {
				log.Printf("There aren't default action for %s", config.Template)
				continue
			}
		}

		res = append(res, config)
	}

	dd := duplicates(res)
	if len(dd) > 0 {
		log.Fatalf("Found duplicates of config names: %s.", strings.Join(dd, ", "))
	}

	return res
}

func removeAfterColon(input string) string {
	if idx := strings.Index(input, ":"); idx != -1 {
		return input[:idx]
	}
	return input // Вернем оригинал, если ":" нет
}

func duplicates(items []contract.ImageConfig) []string {
	nameCount := make(map[string]int)
	var duplicates []string

	for _, item := range items {
		nameCount[item.Template]++
	}

	for name, count := range nameCount {
		if count > 1 {
			duplicates = append(duplicates, name)
		}
	}

	return duplicates
}
