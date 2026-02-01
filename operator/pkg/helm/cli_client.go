package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLIClient implements Client by shelling out to the `helm` CLI.
//
// This keeps the controller logic testable (via the Client interface) while
// avoiding pulling Helm SDK dependencies into controllers.
type CLIClient struct {
	// HelmBinary defaults to "helm".
	HelmBinary string

	// BaseEnv are extra environment variables passed to `helm`.
	BaseEnv []string
}

func NewCLIClient() *CLIClient {
	return &CLIClient{HelmBinary: "helm"}
}

func (c *CLIClient) InstallOrUpgrade(ctx context.Context, req InstallOrUpgradeRequest) (ReleaseInfo, error) {
	if req.Chart.Source != "" && req.Chart.Source != "repository" {
		return ReleaseInfo{}, fmt.Errorf("unsupported chart.source %q for CLI client", req.Chart.Source)
	}
	if req.Chart.Repository == nil {
		return ReleaseInfo{}, fmt.Errorf("chart.repository is required")
	}
	if strings.TrimSpace(req.Chart.Repository.Name) == "" {
		return ReleaseInfo{}, fmt.Errorf("chart.repository.name is required")
	}
	if strings.TrimSpace(req.Chart.Repository.URL) == "" {
		return ReleaseInfo{}, fmt.Errorf("chart.repository.url is required")
	}
	if len(req.Values.ValuesFrom) > 0 {
		return ReleaseInfo{}, fmt.Errorf("values.valuesFrom is not supported by CLI client yet (use values.inline)")
	}

	args := []string{"upgrade", "--install", req.ReleaseName, req.Chart.Repository.Name, "--namespace", req.Namespace}

	if req.CreateNamespace {
		args = append(args, "--create-namespace")
	}
	if req.Timeout.Duration > 0 {
		args = append(args, "--timeout", req.Timeout.Duration.String())
	}
	// Wait for resources to become ready by default. This matches common expectations
	// for "installed" status.
	args = append(args, "--wait")

	args = append(args, "--repo", req.Chart.Repository.URL)
	if req.Chart.Repository.Version != "" {
		args = append(args, "--version", req.Chart.Repository.Version)
	}

	valuesFile, cleanup, err := c.writeInlineValuesTemp(req.Values.Inline)
	if err != nil {
		return ReleaseInfo{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}

	if _, err := c.run(ctx, args); err != nil {
		return ReleaseInfo{}, err
	}
	return c.getReleaseInfo(ctx, req.ReleaseName, req.Namespace)
}

func (c *CLIClient) Uninstall(ctx context.Context, req UninstallRequest) error {
	args := []string{"uninstall", req.ReleaseName, "--namespace", req.Namespace}
	if req.Timeout.Duration > 0 {
		args = append(args, "--timeout", req.Timeout.Duration.String())
	}
	_, err := c.run(ctx, args)
	return err
}

func (c *CLIClient) Test(ctx context.Context, req TestRequest) (TestResult, error) {
	args := []string{"test", req.ReleaseName, "--namespace", req.Namespace}
	if req.Timeout.Duration > 0 {
		args = append(args, "--timeout", req.Timeout.Duration.String())
	}
	// Always include logs so the UI can show output.
	args = append(args, "--logs")
	if strings.TrimSpace(req.Filter) != "" {
		args = append(args, "--filter", req.Filter)
	}

	out, err := c.run(ctx, args)
	if err != nil {
		return TestResult{Succeeded: false, Logs: []string{out}}, err
	}
	return TestResult{Succeeded: true, Logs: []string{out}}, nil
}

func (c *CLIClient) run(ctx context.Context, args []string) (string, error) {
	// Ensure helm has a writable HOME in the container (the manager runs as nonroot).
	// Also isolate helm state to /tmp so different deployments don't conflict.
	baseDir := "/tmp/steer-helm"
	cfgDir := filepath.Join(baseDir, "config")
	cacheDir := filepath.Join(baseDir, "cache")
	dataDir := filepath.Join(baseDir, "data")

	// Best-effort: if these fail, helm will likely fail anyway and bubble up.
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.MkdirAll(cacheDir, 0o755)
	_ = os.MkdirAll(dataDir, 0o755)

	cmd := exec.CommandContext(ctx, c.HelmBinary, args...)
	cmd.Env = append(os.Environ(), c.BaseEnv...)
	cmd.Env = append(cmd.Env,
		"HOME="+baseDir,
		"HELM_CONFIG_HOME="+cfgDir,
		"HELM_CACHE_HOME="+cacheDir,
		"HELM_DATA_HOME="+dataDir,
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		// include combined output in the error for debugging
		return buf.String(), fmt.Errorf("helm %s failed: %w\n%s", strings.Join(args, " "), err, buf.String())
	}
	return buf.String(), nil
}

func (c *CLIClient) writeInlineValuesTemp(inline string) (string, func(), error) {
	inline = strings.TrimSpace(inline)
	if inline == "" {
		return "", nil, nil
	}
	f, err := os.CreateTemp("", "steer-values-*.yaml")
	if err != nil {
		return "", nil, err
	}
	if _, err := f.WriteString(inline); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type helmListItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Revision  string `json:"revision"`
	Status    string `json:"status"`
}

func (c *CLIClient) getReleaseInfo(ctx context.Context, releaseName, namespace string) (ReleaseInfo, error) {
	// helm list is a stable way to get status + revision as JSON.
	// Note: `--filter` is a regexp.
	filter := fmt.Sprintf("^%s$", regexpEscape(releaseName))
	out, err := c.run(ctx, []string{"list", "--namespace", namespace, "--filter", filter, "-o", "json"})
	if err != nil {
		return ReleaseInfo{}, err
	}
	var items []helmListItem
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return ReleaseInfo{}, fmt.Errorf("failed to parse helm list output: %w\n%s", err, out)
	}
	if len(items) == 0 {
		return ReleaseInfo{}, fmt.Errorf("helm release %q not found in namespace %q", releaseName, namespace)
	}

	// Revision is a string. Parse best-effort.
	var rev int64
	if items[0].Revision != "" {
		// don't fail the whole operation if parsing fails.
		parsed, perr := parseInt64(items[0].Revision)
		if perr == nil {
			rev = parsed
		}
	}
	return ReleaseInfo{Name: items[0].Name, Namespace: items[0].Namespace, Version: rev, Status: items[0].Status}, nil
}

func parseInt64(s string) (int64, error) {
	// strconv.ParseInt is small but we keep a tiny helper to avoid extra imports in this file.
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid int64 %q", s)
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}

func regexpEscape(s string) string {
	// Minimal escaping for helm --filter which uses regexp.
	// Escape any character that isn't alphanumeric, '_' or '-'.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteString("\\")
		b.WriteRune(r)
	}
	return b.String()
}
