package handler

import (
	api "github.com/codiewio/codenire/api/gen"
	"net/http"

	"github.com/codiewio/codenire/internal/images"
)

func (h *Handler) ActionsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeData(w)
		return
	}

	if r.Method == http.MethodPost {
		// TODO:: handle errors
		_ = images.PullImageConfigList(h.Config.BackendURL + "/action/list")
		writeData(w)
		return
	}

	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	return
}

func writeData(w http.ResponseWriter) {
	list := images.ImageTemplateList

	var res api.ActionListResponse
	for _, template := range *list {
		var defaultCfg *api.ActionItemResponse
		defaultWrote := false

		for name, config := range template.Actions {
			isDefault := name == "default" || config.IsDefault

			action := api.ActionItemResponse{
				Id:               config.Id,
				Name:             config.Name,
				ContainerOptions: template.ContainerOptions,
				Template:         template.Template,
				Version:          template.Version,
				Workdir:          template.Workdir,
				Groups:           template.Groups,
				Provider:         template.Provider,

				CompileCmd:    config.CompileCmd,
				DefaultFiles:  config.DefaultFiles,
				IsDefault:     isDefault,
				RunCmd:        config.RunCmd,
				ScriptOptions: config.ScriptOptions,
			}

			if isDefault {
				defaultCfg = &action
				// TODO:: ?!
				defaultCfg.IsDefault = true
				continue
			}

			if defaultCfg != nil && defaultCfg.Id == action.Id {
				defaultWrote = true
				action.IsDefault = true
			}

			res = append(res, action)
		}

		if defaultCfg != nil && !defaultWrote {
			res = append(res, *defaultCfg)
		}
	}

	writeJSONResponse(w, res, http.StatusOK)
}
