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
	"bytes"
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
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	contract "sandbox/api/gen"
	"sandbox/internal"
)

const postgresConfigConnection = "postgres"
const imageTagPrefix = "codenire_play/"
const codenireConfigName = "config.json"
const defaultMemoryLimit = 100 << 20

type StartedContainer struct {
	CId    string
	Image  BuiltImage
	TmpDir string
	DBName string
}

type BuiltImage struct {
	contract.ImageConfig

	imageID *string
	tag     string

	buf bytes.Buffer
}

type ContainerOrchestrator interface {
	Prepare() error
	Boot() error
	GetTemplates() []BuiltImage
	GetContainer(ctx context.Context, id string) (*StartedContainer, error)
	KillAll()
	KillContainer(StartedContainer) error
}

type CodenireOrchestrator struct {
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

func NewCodenireOrchestrator() *CodenireOrchestrator {
	c, err := client.NewClientWithOpts(client.WithVersion("1.41"))
	if err != nil {
		panic("fail on createDB docker client")
	}

	log.Printf("using Docker client version: %s", c.ClientVersion())

	return &CodenireOrchestrator{
		dockerClient:        c,
		imageContainers:     make(map[string]chan StartedContainer),
		numSysWorkers:       runtime.NumCPU(),
		idleContainersCount: *replicaContainerCnt,
		dockerFilesPath:     *dockerFilesPath,
		isolated:            *isolated,
	}
}

func (m *CodenireOrchestrator) Prepare() error {
	templates := parseConfigFiles(m.dockerFilesPath)

	for _, t := range templates {
		err := m.prebuildImage(t, m.dockerFilesPath)
		if err != nil {
			log.Println("Build of template failed", "[Template]", t.Template, "[err]", err)
			continue
		}
	}

	return nil
}

func (m *CodenireOrchestrator) Boot() (err error) {
	//err = m.dockerLogin()
	//if err != nil {
	//	return err
	//}

	pool := pond.NewPool(m.numSysWorkers)
	for idx, img := range m.imgs {
		pool.Submit(func() {
			buildErr := m.buildImage(img, idx)
			if buildErr != nil {
				log.Println("Build of Image failed", "[Image]", img.ImageConfig.Template, "[err]", buildErr)
				return
			}
		})
	}

	pool.StopAndWait()

	m.startContainers()

	return nil
}

func (m *CodenireOrchestrator) GetTemplates() []BuiltImage {
	return m.imgs
}

func (m *CodenireOrchestrator) GetContainer(ctx context.Context, id string) (*StartedContainer, error) {
	select {
	case c := <-m.getContainer(id):
		return &c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *CodenireOrchestrator) KillAll() {
	m.Lock()
	defer m.Unlock()

	m.killSignal = true

	defer func() {
		// TODO:: remove tmp dirs
		m.imageContainers = make(map[string]chan StartedContainer)
		m.killSignal = false
	}()

	ctx := context.Background()
	containers, err := m.dockerClient.ContainerList(ctx, docker.ListOptions{All: true})
	if err != nil {
		log.Printf("Get Container List failed: %s", err)
		return
	}

	pool := pond.NewPool(m.numSysWorkers)
	for i := 0; i < len(containers); i++ {
		i := i
		pool.Submit(func() {
			ct := containers[i]

			if !strings.HasPrefix(ct.Image, imageTagPrefix) {
				return
			}

			fmt.Printf("Stop container %s (imageID: %s)...\n", ct.Names[0], ct.ID)

			timeout := 0
			err = m.dockerClient.ContainerStop(ctx, ct.ID, docker.StopOptions{
				Timeout: &timeout,
			})
			if err != nil {
				log.Printf("Stop container failed %s: %s", ct.ID, err)
				return
			}

			fmt.Printf("Container removed: %s\n", ct.ID)
		})
	}
	pool.StopAndWait()
	log.Println("Killed all images")
}

func (m *CodenireOrchestrator) KillContainer(c StartedContainer) (err error) {
	defer func() {
		m.removeSandboxDB(c.DBName)
	}()

	timeout := 0
	err = m.dockerClient.ContainerStop(context.Background(), c.CId, docker.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *CodenireOrchestrator) prebuildImage(cfg contract.ImageConfig, root string) error {
	if !cfg.Enabled {
		return nil
	}
	tag := fmt.Sprintf("%s%s", imageTagPrefix, cfg.Template)

	buf, err := internal.DirToTar(filepath.Join(root, cfg.Template))
	if err != nil {
		return err
	}

	wd := "/app_tmp"
	if cfg.Workdir == "" {
		cfg.Workdir = wd
	}

	m.imgs = append(m.imgs, BuiltImage{
		ImageConfig: cfg,
		imageID:     nil,
		buf:         buf,
		tag:         tag,
	})

	return nil
}

func (m *CodenireOrchestrator) buildImage(i BuiltImage, idx int) error {
	buildOptions := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		Tags:           []string{i.tag},
		Labels:         map[string]string{},
		SuppressOutput: !*dev,
	}

	buildResponse, err := m.dockerClient.ImageBuild(context.Background(), &i.buf, buildOptions)
	if err != nil {
		return fmt.Errorf("error building Image: %w", err)
	}
	defer func() {
		_ = buildResponse.Body.Close()
	}()

	scanner := bufio.NewScanner(buildResponse.Body)
	for scanner.Scan() {
		if *dev {
			fmt.Println("[DEBUG BUILD]", scanner.Text())
		}
	}

	imageInfo, _, err := m.dockerClient.ImageInspectWithRaw(context.Background(), i.tag)
	if err != nil {
		return fmt.Errorf("error on get image info: %w", err)
	}
	if len(imageInfo.RepoTags) < 1 {
		return fmt.Errorf("tags not found for %s", i.Template)
	}

	m.imgs[idx].imageID = &imageInfo.RepoTags[0]

	return nil
}

func (m *CodenireOrchestrator) runSndContainer(img BuiltImage) (cont *StartedContainer, err error) {
	ctx := context.Background()

	networkMode := network.NetworkNone
	var networkEnvs []string
	if img.IsSupportPackage {
		networkMode = *isolatedNetwork
		networkEnvs = append(
			networkEnvs,
			fmt.Sprintf("HTTP_PROXY=%s", *isolatedGateway),
			fmt.Sprintf("HTTPS_PROXY=%s", *isolatedGateway),
		)
	}

	dbName := ""
	if isPostgresConnected(img) {
		name := fmt.Sprintf("pgdb_%s", internal.RandHex(8))
		dbName = name
		user := fmt.Sprintf("pguser_%s", internal.RandHex(8))
		password := fmt.Sprintf("pgpassword_%s", internal.RandHex(8))

		pgErr := createDB(*isolatedPostgresDSN, name, user, password)
		defer func() {
			if err != nil {
				m.removeSandboxDB(name)
			}
		}()
		if pgErr != nil {
			log.Printf("Create PostgreSQL database failed: %s", pgErr.Error())
			return nil, pgErr
		}

		log.Printf("Created PostgreSQL database %s", name)

		networkEnvs = append(
			networkEnvs,
			fmt.Sprintf("PGHOST=%s", "postgres_host"),
			fmt.Sprintf("PGDATABASE=%s", name),
			fmt.Sprintf("PGUSER=%s", user),
			fmt.Sprintf("PGPASSWORD=%s", password),
		)

		if docker.NetworkMode(networkMode).IsNone() {
			networkMode = *isolatedPostgresNetwork
		}
	}

	hostConfig := &docker.HostConfig{
		Runtime:     m.runtime(),
		AutoRemove:  true,
		NetworkMode: docker.NetworkMode(networkMode),
		Resources: docker.Resources{
			Memory:     int64(*img.ContainerOptions.MemoryLimit),
			MemorySwap: 0,
		},
		ExtraHosts: []string{
			"postgres_host:172.25.0.2",
		},
	}
	if img.imageID == nil {
		return nil, fmt.Errorf("imageId is null")
	}
	containerConfig := &docker.Config{
		Image: *img.imageID,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		Env:   networkEnvs,
	}

	name := stripImageName(*img.imageID)
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
		return nil, fmt.Errorf("create container failed: %w", err)
	}

	err = m.dockerClient.ContainerStart(ctx, containerResp.ID, docker.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("create container failed: %w", err)
	}

	// External connect when networkMode already set up and networkMode not isolatedPostgresNetwork
	if !hostConfig.NetworkMode.IsNone() &&
		isPostgresConnected(img) &&
		networkMode != *isolatedPostgresNetwork {
		err = m.dockerClient.NetworkConnect(ctx, *isolatedPostgresNetwork, containerResp.ID, &network.EndpointSettings{})
		if err != nil {
			return nil, err
		}
	}

	return &StartedContainer{
		CId:    containerResp.ID,
		Image:  img,
		DBName: dbName,
	}, nil
}

func isPostgresConnected(img BuiltImage) bool {
	pgConnected := false
	for _, c := range img.Connections {
		if c == postgresConfigConnection {
			pgConnected = true
		}
	}

	return *isolatedPostgresDSN != "" &&
		*isolatedPostgresNetwork != "" &&
		pgConnected
}

func (m *CodenireOrchestrator) startContainers() {
	var ii []string
	for _, img := range m.imgs {
		ii = append(ii, img.Template)
	}
	log.Printf("Starting images: %s", strings.Join(ii, ","))

	for _, img := range m.imgs {
		for i := 0; i < m.idleContainersCount; i++ {
			go func() {
				for {
					if m.killSignal {
						continue
					}

					c, err := m.runSndContainer(img)
					if err != nil {
						log.Printf("[DEBUG] Run container error. Template: %s. Error: %s", img.Template, err.Error())
						time.Sleep(10 * time.Second)
						continue
					}

					m.getContainer(img.Template) <- *c
				}
			}()
		}
	}
}

func (m *CodenireOrchestrator) getContainer(template string) chan StartedContainer {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.imageContainers[template]; !exists {
		m.imageContainers[template] = make(chan StartedContainer)
	}

	return m.imageContainers[template]
}

func (m *CodenireOrchestrator) runtime() string {
	if m.isolated {
		return "runsc"
	}

	return ""
}

func (m *CodenireOrchestrator) removeSandboxDB(dbname string) {
	if *isolatedPostgresDSN != "" && dbname != "" {
		// TODO:: handle it and cover by prometheus
		_ = dropDB(*isolatedPostgresDSN, dbname)
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

// nolint
func parseConfigFiles(root string) []contract.ImageConfig {
	directories := internal.ListDirectories(root)

	var res []contract.ImageConfig

	for _, d := range directories {
		dir := filepath.Join(root, d)

		info, err := os.Stat(dir)
		if err != nil {
			log.Printf("err1", err)
			continue
		}

		if !info.IsDir() {
			log.Printf("not dir", err)
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
			_, defaultExists := config.Actions[DefaultActionName]
			var first *contract.ImageActionConfig

			for n, actionConfig := range config.Actions {
				if first == nil {
					first = &actionConfig
				}

				// Handle defaults enable commands
				if actionConfig.EnableExternalCommands == "" {
					actionConfig.EnableExternalCommands = ExternalCommandsModeAll
					config.Actions[n] = actionConfig
				}

				// Handle default action
				if actionConfig.IsDefault && !defaultExists {
					defaultExists = true
					actionConfig.IsDefault = true
					config.Actions[DefaultActionName] = actionConfig
					continue
				}
			}

			if first != nil && !defaultExists && first.Name != "" {
				config.Actions[DefaultActionName] = *first
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
	return input
}

func duplicates(items []contract.ImageConfig) []string {
	nameCount := make(map[string]int)
	var dd []string

	for _, item := range items {
		nameCount[item.Template]++
	}

	for name, count := range nameCount {
		if count > 1 {
			dd = append(dd, name)
		}
	}

	return dd
}

//nolint
//func (m *CodenireOrchestrator) dockerLogin() error {
//	dockerUsername := os.Getenv("DOCKER_USERNAME")
//	dockerToken := os.Getenv("DOCKER_LOGIN_TOKEN")
//
//	if dockerToken == "" || dockerUsername == "" {
//		return fmt.Errorf("docker credentials not provided")
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	authConfig := registry.AuthConfig{
//		Username: dockerUsername,
//		Password: dockerToken,
//	}
//
//	authResponse, err := m.dockerClient.RegistryLogin(ctx, authConfig)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println("Docker  Login successful:", authResponse.Status)
//	return nil
//}
