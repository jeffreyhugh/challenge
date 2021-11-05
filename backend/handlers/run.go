package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/archive"
	"github.com/gorilla/mux"
	"github.com/lucsky/cuid"
	"github.com/qbxt/gologger"
	"github.com/sirupsen/logrus"
	"queue.bot/challenge/constants"
	"queue.bot/challenge/structs"
)

type runResp struct {
	ID       string `json:"id"`
	Language string `json:"language"`
}

type runError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

/**
 * Run takes a language and a code block, then returns the ID of the execution instance
 * The client is supposed to check on the ID to see the status of the running request
 */
func HandleRun(w http.ResponseWriter, r *http.Request, cr *structs.CustomRouter) {
	vars := mux.Vars(r)
	selectedLang := ""

	for _, lang := range constants.Languages {
		if strings.ToLower(vars["language"]) == lang {
			selectedLang = lang
		}
	}

	if selectedLang == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(runError{
			Code:    http.StatusBadRequest,
			Message: "no language provided",
		})
		return
	}

	id := cuid.New()
	gologger.Debug("new image id", logrus.Fields{"id": id})
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(runError{
			Code:    http.StatusInternalServerError,
			Message: "could not read body",
		})
		gologger.Warn("could not read body", err, nil)
		return
	}

	if err := os.WriteFile(fmt.Sprintf("submissions/%s.%s", id, selectedLang), []byte(body), 0440); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(runError{
			Code:    http.StatusInternalServerError,
			Message: "could not save code",
		})
		gologger.Warn("could not save code", err, nil)
		return
	}

	go func() {
		ctx := context.Background()
		dockerBuildContext, _ := archive.TarWithOptions("./", &archive.TarOptions{
			IncludeFiles: []string{
				fmt.Sprintf("dockerfiles/%s.dockerfile", selectedLang),
				fmt.Sprintf("submissions/%s.%s", id, selectedLang),
			},
		})
		_, err = cr.Docker.ImageBuild(ctx, dockerBuildContext, types.ImageBuildOptions{
			Dockerfile: fmt.Sprintf("dockerfiles/%s.dockerfile", selectedLang),
			BuildArgs: map[string]*string{
				"SUBMISSION_ID": &id,
			},
			Tags:        []string{fmt.Sprintf("challenge/%s", id)},
			ForceRemove: true,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(runError{
				Code:    http.StatusInternalServerError,
				Message: "could not build image",
			})
			gologger.Warn("could not build image", err, nil)
			return
		}

		// create the container
		resp := &container.ContainerCreateCreatedBody{}
		for i := 0; i < 32; i++ {
			r, err := cr.Docker.ContainerCreate(ctx, &container.Config{
				Image: fmt.Sprintf("challenge/%s", id),
			}, nil, nil, nil, id)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
			} else {
				gologger.Debug("built container", logrus.Fields{
					"id":    id,
					"tries": i,
				})
				resp = &r
				break
			}
		}

		if resp.ID == "" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(runError{
				Code:    http.StatusInternalServerError,
				Message: "could not build container",
			})
			gologger.Warn("could not build container", err, nil)
			return
		}

		// start the container
		if err := cr.Docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(runError{
				Code:    http.StatusInternalServerError,
				Message: "could not start container",
			})
			gologger.Warn("could not start container", err, nil)
			return
		}
	}()

	json.NewEncoder(w).Encode(runResp{
		ID:       id,
		Language: selectedLang,
	})
}
