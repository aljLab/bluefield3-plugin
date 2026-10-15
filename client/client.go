package client

import (
	"bytes"
	"fmt"
	"net/http"
	"io"
	"os"
	"encoding/json"
	"path/filepath"
	"strings"
)

type Result struct {
	DplConfigPath   string
	P4InfoPath      string
}

type DplClient struct {
	ServerAddr       string
	ServerPort       string
	VolumePath       string
	progs map[string]string
}

type DplCompileRequest struct {
	Target      string
	FileName string
}

type DplLoadRequest struct {
	DeviceID    string
	GrpcAddr    string
	FileName string
}

func NewDplClient(serverAddr, serverPort, volumePath string) *DplClient {
	return &DplClient{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		VolumePath: volumePath,
		progs: make(map[string]string),
	}
}

// fetches the program from a remote URL to host-shared volume (accessible for DPL)
// creates a subdir /tmp/loom/<programName> to store compilation artifacts in
// stores the program dir on the DplClient for cleanup
func (c *DplClient) FetchProgram(fileURL string) error {
	fileName := filepath.Base(fileURL)
	programName := strings.TrimSuffix(fileName, filepath.Ext(fileURL))

	fmt.Println("creating subdirectory for program")

	dir := fmt.Sprintf("%s/%s", c.VolumePath, programName)
	err := os.MkdirAll(dir, 0755); if err != nil {
		return fmt.Errorf("creating subdirectory %s", err)
	}


	resp, err := http.Get(fileURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("downloading %q: %w", fileURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("downloading %q: HTTP %d", fileURL, resp.StatusCode)
	}

	path := fmt.Sprintf("%s/%s", dir, fileName)
	fmt.Println("creating file at ", path)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	c.progs[programName] = dir

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func (c *DplClient) CompileP4File(fileName string) error {
	req := DplCompileRequest{
		Target: "doca-bf3",
		FileName: fileName,
	}

	_, err := c.postCompileRequest(req); if err != nil {
		return fmt.Errorf("compiling file: %w", err)
	}

	fmt.Println("Compilation successful.")
	return nil
}

func (c *DplClient) LoadP4RuntimeShell(fileName string) error {
	req := DplLoadRequest{
		DeviceID: "1000",
		GrpcAddr: "172.16.44.81:9559",
		FileName: fileName,
	}

	_, err := c.postLoadRequest(req); if err != nil {
		return fmt.Errorf("loadig file: %w", err)
	}

	fmt.Println("Loading successful.")
	return nil
}

// request compilation of a p4 program located in the shared volume by name
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

// request DPL to load a p4 program onto the BlueField
// artifacts need to be compiled prior and are located in the shared volume
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

// clean up a subdirectory associated with a nf in the shared volume
func (c *DplClient) Cleanup(programName string) error {
	dir, ok := c.progs[programName]; if !ok {
		return fmt.Errorf("Did not find program %s to clean up", programName)
	}

	os.RemoveAll(dir)
	return nil
}
