package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) IsDockerAvailable() bool {
	if c.cli == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.cli.Ping(ctx)
	return err == nil
}

func (c *Client) ListContainers(ctx context.Context, all bool) ([]container.Summary, error) {
	return c.cli.ContainerList(ctx, container.ListOptions{All: all})
}

func (c *Client) GetContainer(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return c.cli.ContainerInspect(ctx, containerID)
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	return c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) GetSystemInfo(ctx context.Context) (system.Info, error) {
	return c.cli.Info(ctx)
}

// Image methods
func (c *Client) ListImages(ctx context.Context, all bool) ([]image.Summary, error) {
	return c.cli.ImageList(ctx, image.ListOptions{All: all})
}

func (c *Client) GetImage(ctx context.Context, id string) (image.InspectResponse, error) {
	return c.cli.ImageInspect(ctx, id)
}

func (c *Client) RemoveImage(ctx context.Context, id string, force bool, noPrune bool) ([]image.DeleteResponse, error) {
	options := image.RemoveOptions{
		Force:         force,
		PruneChildren: !noPrune,
	}

	return c.cli.ImageRemove(ctx, id, options)
}

func (c *Client) PullImage(ctx context.Context, fromImage string, tag string, platform string) error {
	pullOptions := image.PullOptions{
		Platform: platform,
	}

	imageRef := fromImage
	if tag != "" {
		imageRef = fmt.Sprintf("%s:%s", fromImage, tag)
	}

	reader, err := c.cli.ImagePull(ctx, imageRef, pullOptions)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Read the response to ensure the pull completes
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	return nil
}

// New method for streaming pull
func (c *Client) PullImageStream(ctx context.Context, fromImage string, tag string, platform string) (io.ReadCloser, error) {
	pullOptions := image.PullOptions{
		Platform: platform,
	}

	imageRef := fromImage
	if tag != "" {
		imageRef = fmt.Sprintf("%s:%s", fromImage, tag)
	}

	reader, err := c.cli.ImagePull(ctx, imageRef, pullOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	return reader, nil
}

func (c *Client) PullImageWithStream(ctx context.Context, imageName string, writer io.Writer) error {
	pullOptions := image.PullOptions{}

	reader, err := c.cli.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Stream the response directly to the writer
	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to stream pull response: %w", err)
	}

	return nil
}

func (c *Client) BuildImage(ctx context.Context, contextPath string, dockerfile string, tags []string, buildArgs map[string]string, target string, platform string) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you'd need to create a tar archive of the build context
	// and handle the build response stream properly
	return "", fmt.Errorf("build image not implemented yet")
}

func (c *Client) TagImage(ctx context.Context, source string, repository string, tag string) error {
	targetRef := repository
	if tag != "" {
		targetRef = fmt.Sprintf("%s:%s", repository, tag)
	}

	return c.cli.ImageTag(ctx, source, targetRef)
}

func (c *Client) PushImage(ctx context.Context, imageID string, tag string) error {
	pushRef := imageID
	if tag != "" {
		pushRef = fmt.Sprintf("%s:%s", imageID, tag)
	}

	pushOptions := image.PushOptions{}

	reader, err := c.cli.ImagePush(ctx, pushRef, pushOptions)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer reader.Close()

	// Read the response to ensure the push completes
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read push response: %w", err)
	}

	return nil
}

// Network methods
func (c *Client) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	return c.cli.NetworkList(ctx, network.ListOptions{})
}

func (c *Client) GetNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	return c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{})
}

func (c *Client) CreateNetwork(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return c.cli.NetworkCreate(ctx, name, options)
}

func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	return c.cli.NetworkRemove(ctx, networkID)
}

func (c *Client) ConnectContainerToNetwork(ctx context.Context, networkID string, containerID string, config *network.EndpointSettings) error {
	return c.cli.NetworkConnect(ctx, networkID, containerID, config)
}

func (c *Client) DisconnectContainerFromNetwork(ctx context.Context, networkID string, containerID string, force bool) error {
	return c.cli.NetworkDisconnect(ctx, networkID, containerID, force)
}

func (c *Client) PruneNetworks(ctx context.Context) (network.PruneReport, error) {
	filterArgs := filters.NewArgs()
	return c.cli.NetworksPrune(ctx, filterArgs)

}

// Volume methods
func (c *Client) ListVolumes(ctx context.Context) (volume.ListResponse, error) {
	return c.cli.VolumeList(ctx, volume.ListOptions{})
}

func (c *Client) GetVolume(ctx context.Context, volumeID string) (volume.Volume, error) {
	return c.cli.VolumeInspect(ctx, volumeID)
}

func (c *Client) CreateVolume(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return c.cli.VolumeCreate(ctx, options)
}

func (c *Client) RemoveVolume(ctx context.Context, volumeID string, force bool) error {
	return c.cli.VolumeRemove(ctx, volumeID, force)
}

func (c *Client) PruneVolumes(ctx context.Context) (volume.PruneReport, error) {
	filterArgs := filters.NewArgs()
	return c.cli.VolumesPrune(ctx, filterArgs)
}

func (c *Client) PruneVolumesWithFilters(ctx context.Context, filterArgs filters.Args) (volume.PruneReport, error) {
	return c.cli.VolumesPrune(ctx, filterArgs)
}

func (c *Client) GetVolumeUsage(ctx context.Context, name string) (bool, []string, error) {
	if _, err := c.cli.VolumeInspect(ctx, name); err != nil {
		return false, nil, fmt.Errorf("volume not found: %w", err)
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return false, nil, fmt.Errorf("failed to list containers: %w", err)
	}

	inUse := false
	var usingContainers []string

	for _, container := range containers {
		containerInfo, err := c.cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			continue
		}

		for _, mount := range containerInfo.Mounts {
			if mount.Type == "volume" && mount.Name == name {
				inUse = true
				usingContainers = append(usingContainers, container.ID)
				break
			}
		}
	}

	return inUse, usingContainers, nil
}

func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}
