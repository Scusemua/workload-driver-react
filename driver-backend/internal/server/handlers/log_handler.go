package handlers

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

var (
	ErrMissingPod       = errors.New("request did not specify pod name")
	ErrMissingContainer = errors.New("request did not specify container name")
)

type LogHttpHandler struct {
	*BaseHandler
}

func NewLogHttpHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *LogHttpHandler {
	handler := &LogHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side LogHttpHandler.")

	return handler
}

func (h *LogHttpHandler) HandleRequest(c *gin.Context) {
	pod := c.Param("pod")
	container := c.Query("container")
	follow := c.Query("follow")

	var doFollow = false
	if follow == "true" {
		doFollow = true
	}

	h.logger.Debug("Received log request.", zap.String("pod", pod), zap.String("container", container))

	if pod == "" {
		h.logger.Error("Log request is missing the pod argument.")
		_ = c.AbortWithError(http.StatusBadRequest, ErrMissingPod)
		return
	} else if container == "" {
		h.logger.Error("Log request is missing the container argument.", zap.String("pod", pod))
		_ = c.AbortWithError(http.StatusBadRequest, ErrMissingContainer)
		return
	}

	url := fmt.Sprintf("http://localhost:8889/api/v1/namespaces/default/pods/%s/log?container=%s&follow=%v", pod, container, doFollow)
	h.logger.Debug("Retrieving logs now.", zap.String("pod", pod), zap.String("container", container), zap.String("url", url))
	resp, err := http.Get(url)
	if err != nil {
		h.logger.Error("Failed to get logs.", zap.String("pod", pod), zap.String("container", container), zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Failed to retrieve logs.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status))
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = c.AbortWithError(resp.StatusCode, fmt.Errorf("failed to retrieve logs: received HTTP %d %s", resp.StatusCode, resp.Status))
			return
		} else {
			_ = c.AbortWithError(resp.StatusCode, fmt.Errorf("failed to retrieve logs (received HTTP %d %s): %s", resp.StatusCode, resp.Status, payload))
			return
		}
	}

	if doFollow {
		h.streamLogs(c, resp, pod, container)
	} else {
		h.logger.Debug("Sending all logs back to client at once (i.e., not streaming them).", zap.String("pod", pod), zap.String("container", container))
		resp, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusOK)
		bytesWritten, err := c.Writer.Write(resp)
		if err != nil {
			h.logger.Error("Failed to write logs back to client.", zap.String("pod", pod), zap.String("container", container), zap.Error(err))
		} else {
			h.sugaredLogger.Debugf("Wrote %d bytes back to client for logs of container %s of pod %s.", bytesWritten, container, pod)
		}
		return
	}
}

func (h *LogHttpHandler) streamLogs(c *gin.Context, resp *http.Response, pod string, container string) {
	h.logger.Debug("Streaming logs to client.", zap.String("pod", pod), zap.String("container", container))
	c.Header("Transfer-Encoding", "chunked")

	streamChan := make(chan []byte)
	go func() {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			streamChan <- line
		}
	}()

	var messagesSent = 0
	var printEvery = 100
	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-streamChan; ok {
			_, err := w.Write(msg)
			if err != nil {
				h.logger.Error("Error while writing stream response for logs.", zap.String("pod", pod), zap.String("container", container), zap.Error(err))
			}
			c.Writer.Flush()

			messagesSent += 1

			if messagesSent%printEvery == 0 {
				h.sugaredLogger.Debugf("Transmitted %d messages of logs for container %s of pod %s", messagesSent, container, pod)
				printEvery = 500
			}
			return true // Keep open
		}

		h.logger.Error("Client disconnected in the middle of the stream.", zap.String("pod", pod), zap.String("container", container))
		return false // Close stream
	})
}
