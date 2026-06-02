package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/shiiit/micro-cloud/internal/database"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WorkspaceTerminal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "workspace id required", http.StatusBadRequest)
		return
	}

	var containerID string
	row := database.QueryRow(`SELECT container_id FROM workspaces WHERE id = ?`, id)
	if err := row.Scan(&containerID); err != nil || containerID == "" {
		http.Error(w, "workspace not found or no container", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TERM] Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("[TERM] Docker client: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("docker client error"))
		return
	}
	defer cli.Close()

	execConfig := types.ExecConfig{
		Cmd:          []string{"sh"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}

	execResp, err := cli.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		log.Printf("[TERM] Exec create: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("exec create error: %v", err)))
		return
	}

	startCheck := types.ExecStartCheck{Tty: true, Detach: false}
	attachResp, err := cli.ContainerExecAttach(context.Background(), execResp.ID, startCheck)
	if err != nil {
		log.Printf("[TERM] Exec attach: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("attach error: %v", err)))
		return
	}
	defer attachResp.Close()

	err = cli.ContainerExecStart(context.Background(), execResp.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		log.Printf("[TERM] Exec start: %v", err)
		return
	}

	errCh := make(chan error, 2)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := attachResp.Reader.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n > 0 {
				msg := buf[:n]
				if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			attachResp.Conn.Write(msg)
		}
	}()

	select {
	case <-errCh:
	case <-time.After(30 * time.Minute):
	}

	log.Printf("[TERM] Workspace %s terminal closed", id)
}
