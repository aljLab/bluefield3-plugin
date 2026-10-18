package p4compile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	// "time"

	oldproto "github.com/golang/protobuf/proto"
	p4configv1 "github.com/p4lang/p4runtime/go/p4/config/v1"
)

// Result holds the output of a successful P4 compilation.
type Result struct {
	DeviceConfig []byte
	P4Info       *p4configv1.P4Info
}

func LoadCompiledArchive(ctx context.Context, fileURL string) (*Result, error) {
	resp, err := http.Get(fileURL) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("downloading %q: %w", fileURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("downloading %q: HTTP %d", fileURL, resp.StatusCode)
	}

	// unpacks archive and returns file names
	dplConfigName, p4InfoName, err := unpackArchive(resp); if err != nil {
		return nil, fmt.Errorf("unpacking archive: %w", err)
	}
	
	defer cleanUp()
	
	deviceConfigBytes, err := os.ReadFile(fmt.Sprintf("/tmp/unpack/%s", dplConfigName));if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dplConfigName, err)
	}
	p4infoBytes, err := os.ReadFile(fmt.Sprintf("/tmp/unpack/%s", p4InfoName));if err != nil {
		return nil, fmt.Errorf("reading p4info: %w", err)
	}

	var p4info p4configv1.P4Info
	if err := oldproto.Unmarshal(p4infoBytes, &p4info); err != nil {
		return nil, fmt.Errorf("parsing p4info: %w", err)
	}

	return &Result{
		DeviceConfig: deviceConfigBytes,
		P4Info: &p4info,
	}, nil
}

func unpackArchive(resp *http.Response)(string, string, error){
	err := os.MkdirAll("/tmp/unpack", 0755); if err != nil {
		return "", "", fmt.Errorf("while creating /tmp/unpack: %w", err)
	}

	f, err := os.Create("/tmp/unpack/p4archive.tar.gz")

	if err != nil {
		return "", "", fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", "", fmt.Errorf("writing archive file: %w", err)
	}
	
	cmd := exec.Command(
		"tar", "-xzf",
		"/tmp/unpack/p4archive.tar.gz",
		"-C", "/tmp/unpack/",
	)
	out, err := cmd.CombinedOutput(); if err != nil {
		return "", "", fmt.Errorf("unpacking archive %w, %s", err, string(out))
	}

	entries, err := os.ReadDir("/tmp/unpack"); if err != nil {
		return "", "", fmt.Errorf("while reading /tmp/unpack: %w", err)
	}

	var foundDplConfig, foundP4Info bool
	var dplConfigName, p4InfoName string

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".dplconfig") {
			dplConfigName = entry.Name()
			foundDplConfig = true
			break
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".p4info.txt") {
			p4InfoName = entry.Name()
			foundP4Info = true
			break
		}
	}

	if !foundDplConfig || !foundP4Info {
		return "", "", fmt.Errorf("Couldn't unpack archive. Make sure to provide a flat archive without nested directories, e.g. produced by 'tar -czf archive.tar.gz *.dplconfig *.p4info.txt'")
	}
	
	return dplConfigName, p4InfoName, nil
}

func cleanUp(){
	err := os.RemoveAll("/tmp/unpack/"); if err != nil {
		fmt.Printf("Failed cleaning up: removing /tmp/unpack/*: %w", err)
	}
}

/* // CompileFromURL downloads a .p4 source file from fileURL and compiles it with
// p4c for the bmv2/v1model target.
func CompileFromURL(ctx context.Context, fileURL string) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "p4compile-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := tmpDir + "/input.p4"
	if err := downloadFile(fileURL, inputPath); err != nil {
		return nil, err
	}
	return compileP4File(ctx, inputPath, tmpDir)
}

func downloadFile(fileURL, destPath string) error {
	resp, err := http.Get(fileURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("downloading %q: %w", fileURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("downloading %q: HTTP %d", fileURL, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

func compileP4File(ctx context.Context, inputPath, outDir string) (*Result, error) {
	p4infoPath := outDir + "/p4info.bin"

	compileCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, "p4c",
		"--target", "bmv2",
		"--arch", "v1model",
		"--p4runtime-files", p4infoPath,
		"--p4runtime-format", "binary",
		"-o", outDir,
		inputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("p4c: %s", string(out))
	}

	jsonBytes, err := os.ReadFile(outDir + "/input.json")
	if err != nil {
		return nil, fmt.Errorf("reading compiled JSON: %w", err)
	}

	p4infoBytes, err := os.ReadFile(p4infoPath)
	if err != nil {
		return nil, fmt.Errorf("reading p4info: %w", err)
	}

	var p4info p4configv1.P4Info
	if err := oldproto.Unmarshal(p4infoBytes, &p4info); err != nil {
		return nil, fmt.Errorf("parsing p4info: %w", err)
	}

	return &Result{
		DeviceConfig: jsonBytes,
		P4Info:       &p4info,
	}, nil
} */