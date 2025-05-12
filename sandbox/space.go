package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	contract "sandbox/api/gen"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

var (
	spaceManager SpaceManager
	basePath     = "/space"
)

type Space struct {
	Version int
}

type SpaceManager struct {
	Cache *expirable.LRU[string, Space]
}

func NewSpaceManager() *SpaceManager {
	cache := expirable.NewLRU[string, Space](1024, func(id string, value Space) {
		fullPath := filepath.Join(basePath, id)
		err := os.RemoveAll(fullPath)
		if err != nil {
			fmt.Printf("Ошибка при удалении: %v\n", err)
		}
	}, time.Hour*24)

	// TODO:: удалить старые (больше 24 часов) spaces

	return &SpaceManager{
		Cache: cache,
	}
}

func (s *SpaceManager) GetOrCreate(id string) (it *Space, err error) {
	defer func() {
		if err != nil {
			fmt.Printf("errorrrrr : %v", err)
			s.Cache.Remove(id)
		}
	}()

	fullPath := filepath.Join(basePath, id)

	ex, ok := s.Cache.Get(id)
	if !ok {
		newSpace := Space{
			Version: 0,
		}

		ex = newSpace
		s.Cache.Add(id, newSpace)
	}

	{
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			err = os.Mkdir(fullPath, 0755)
			if err != nil {
				return nil, fmt.Errorf("Ошибка при создании папки: %v\n", err)
			}
		}
	}

	return &ex, nil
}

func (s *SpaceManager) sessionConnectHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var req contract.StartSessionRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	space, err := s.GetOrCreate(req.Id)
	if err != nil {
		http.Error(w, "SpaceManager create failed", http.StatusConflict)
		return
	}

	res := &contract.StartSessionResponse{
		Cluster: req.Cluster,
		Version: &space.Version,
	}

	body, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
