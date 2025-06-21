package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ofkm/arcane-agent/internal/docker"
)

type ContainerService struct {
	dockerClient *docker.Client
}

func NewContainerService(dockerClient *docker.Client) *ContainerService {
	return &ContainerService{
		dockerClient: dockerClient,
	}
}

func (s *ContainerService) GetStats(ctx context.Context, containerID string, stream bool) (interface{}, error) {
	stats, err := s.dockerClient.ContainerStats(ctx, containerID, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var statsData interface{}
	decoder := json.NewDecoder(stats.Body)
	if err := decoder.Decode(&statsData); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return statsData, nil
}

func (s *ContainerService) StreamStats(ctx context.Context, containerID string, statsChan chan<- interface{}) error {
	stats, err := s.dockerClient.ContainerStats(ctx, containerID, true)
	if err != nil {
		return fmt.Errorf("failed to start stats stream: %w", err)
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var statsData interface{}
			if err := decoder.Decode(&statsData); err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("failed to decode stats: %w", err)
			}

			select {
			case statsChan <- statsData:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
