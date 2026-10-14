package nf

import (
	"context"
	"fmt"
	"os/exec"
	"time"
	"path/filepath"
)

type Result struct {
	DplConfigPath   string
	P4InfoPath      string
}

func compileP4File(ctx context.Context, inputPath, outDir string) (*Result, error) {
	p4FileName := filepath.Base(inputPath)
	dplConfigPath := fmt.Sprintf("%s/%s.dplconfig", outDir, p4FileName)
	p4infoPath := fmt.Sprintf("%s/%s.p4info.txt", outDir, p4FileName)

	compileCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, "dpl compile",
		"--target", "doca-bf3",
		inputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("nvp4c: %s", string(out))
	}

	return &Result{
		DplConfigPath: dplConfigPath,
		P4InfoPath:       p4infoPath,
	}, nil
}

func loadP4RuntimeShell(ctx context.Context, res *Result) error {

	cmd := exec.CommandContext(ctx, "dpl compile",
		"--grpc-addr", "172.16.44.81:9559",
		"--device-id", "1000",
		"--config", fmt.Sprintf("%s,%s", res.P4InfoPath, res.DplConfigPath),
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nvp4c: %s", string(out))
	}

	return nil
}
