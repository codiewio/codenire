package images

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	api "github.com/codiewio/codenire/api/gen"
	"github.com/codiewio/codenire/internal/client"
)

var ExtendedTemplates []api.ImageConfig
var ImageTemplateList *[]api.ImageConfig

const templatesPath = "templates"

func PullImageConfigList(url string) (res *[]api.ImageConfig, err error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url+"/"+templatesPath,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("sandbox client request error: %w", err)
	}

	resp, err := client.SandboxBackendClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sandbox client request error: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("response status: %v", resp)

	var execRes []api.ImageConfig
	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return nil, err
	}

	execRes = append(execRes, ExtendedTemplates...)

	return &execRes, nil
}

func GetImageConfig(templateID string) *api.ImageConfig {
	configs := *ImageTemplateList
	for _, config := range configs {
		if config.Template == templateID {
			return &config
		}
	}

	return nil
}
