package images

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/codiewio/codenire/api/gen"
	"github.com/codiewio/codenire/internal/client"
	"net/http"
)

var ConfigList *api.ImageConfigList

func PullImageConfigList(url string) error {
	req, err := http.NewRequestWithContext(
		context.Background(),
		"GET",
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

	var execRes api.ImageConfigList

	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return err
	}

	ConfigList = &execRes

	return nil
}
