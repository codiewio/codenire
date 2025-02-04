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

func PullImageConfigList(url string) error {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return fmt.Errorf("sandbox client request error: %w", err)
	}

	resp, err := client.SandboxBackendClient().Do(req)
	if err != nil {
		return fmt.Errorf("sandbox client request error: %w", err)
	}
	defer resp.Body.Close()

	var execRes []api.ImageConfig
	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return err
	}

	execRes = append(execRes, ExtendedTemplates...)
	ImageTemplateList = &execRes

	log.Printf("images config list data refreshed")

	return nil
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
