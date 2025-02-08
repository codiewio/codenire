package handler

import (
	"net/http"

	api "github.com/codiewio/codenire/api/gen"
	"github.com/codiewio/codenire/internal/images"
)

const defaultAction = "default"

func (h *Handler) ActionListHandler(w http.ResponseWriter, _ *http.Request) {
	list, err := images.PullImageConfigList(h.Config.BackendURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var res api.ActionListResponse
	for _, template := range *list {
		var defaultCfg *api.ActionItemResponse
		defaultWrote := false

		for name, config := range template.Actions {
			isDefault := name == defaultAction || config.IsDefault

			action := api.ActionItemResponse{
				Id:               config.Id,
				Name:             config.Name,
				ContainerOptions: template.ContainerOptions,
				Template:         template.Template,
				Version:          template.Version,
				Workdir:          template.Workdir,
				Groups:           template.Groups,
				Provider:         template.Provider,

				CompileCmd:             config.CompileCmd,
				DefaultFiles:           config.DefaultFiles,
				IsDefault:              isDefault,
				RunCmd:                 config.RunCmd,
				ScriptOptions:          config.ScriptOptions,
				EnableExternalCommands: api.ActionItemResponseEnableExternalCommands(config.EnableExternalCommands),
			}

			if isDefault {
				defaultCfg = &action
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
