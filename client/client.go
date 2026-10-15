package client

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"io"
	"os"
	"encoding/json"
	"path/filepath"
)

type Result struct {
	DplConfigPath   string
	P4InfoPath      string
}

type DplClient struct {
	ServerAddr    string
	ServerPort    string
	VolumePath    string
}

type DplCompileRequest struct {
	Target      string
	ProgramName string
}

type DplLoadRequest struct {
	DeviceID    string
	GrpcAddr    string
	ProgramName string
	Config      string
}

func NewDplClient(serverAddr, serverPort, volumePath string) *DplClient {
	return &DplClient{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		VolumePath: volumePath,
	}
}

func (c *DplClient) GetConfigString(fileName string) (string, error) {
	if fileName == "" || filepath.Ext(fileName) != ".p4" {
		return "", errors.New("file name can not be empty and has to be *.p4")
	}

	return fmt.Sprintf("%s/%s.p4info.txt,%s/%s.dplconfig", c.VolumePath, fileName, c.VolumePath, fileName), nil
}

func (c *DplClient) FetchProgram(fileURL string) error {

	resp, err := http.Get(fileURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("downloading %q: %w", fileURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("downloading %q: HTTP %d", fileURL, resp.StatusCode)
	}

	f, err := os.Create(c.VolumePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

func (c *DplClient) CompileP4File(fileName string) error {
	req := DplCompileRequest{
		Target: "doca-bf3",
		ProgramName: fileName,
	}

	_, err := c.postCompileRequest(req); if err != nil {
		return fmt.Errorf("compiling file: %w", err)
	}

	fmt.Println("Compilation successful.")
	return nil
}

func (c *DplClient) LoadP4RuntimeShell(fileName string) error {
	loadConfig, err := c.GetConfigString(fileName); if err != nil {
		return fmt.Errorf("formatting load config: %w", err)
	}

	req := DplLoadRequest{
		DeviceID: "1000",
		GrpcAddr: "172.16.44.81:9559",
		ProgramName: fileName,
		Config: loadConfig,
	}

	_, err = c.postLoadRequest(req); if err != nil {
		return fmt.Errorf("loadig file: %w", err)
	}

	fmt.Println("Loading successful.")
	return nil
}

func (c *DplClient) postCompileRequest(body DplCompileRequest) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	res, err := http.Post(
		fmt.Sprintf("http://%s:%s/dpl/compile", c.ServerAddr, c.ServerPort),
		"application/json", bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-2XX response: %s", res.Status)
	}
	
	return res, nil
}

func (c *DplClient) postLoadRequest(body DplLoadRequest) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	res, err := http.Post(
		fmt.Sprintf("http://%s:%s/dpl/load", c.ServerAddr, c.ServerPort),
		"application/json", bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-2XX response: %s", res.Status)
	}
	
	return res, nil
}

