package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
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
	// default to repository for backwards compatibility.
	source := strings.TrimSpace(string(req.Chart.Source))
	if source == "" {
		source = "repository"
	}
	if len(req.Values.ValuesFrom) > 0 {
		return ReleaseInfo{}, fmt.Errorf("values.valuesFrom is not supported by CLI client yet (use values.inline)")
	}

	chartRef, cleanup, err := c.resolveChartRef(ctx, req.Chart)
	if err != nil {
		return ReleaseInfo{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// If chart.source isn't explicitly repository, ensure resolveChartRef produced a local ref.
	if source != "repository" && chartRef.Kind != chartRefKindLocal {
		return ReleaseInfo{}, fmt.Errorf("unsupported chart.source %q for CLI client", req.Chart.Source)
	}

	args := []string{"upgrade", "--install", req.ReleaseName, chartRef.Ref, "--namespace", req.Namespace}

	if req.CreateNamespace {
		args = append(args, "--create-namespace")
	}
	if req.Timeout.Duration > 0 {
		args = append(args, "--timeout", req.Timeout.Duration.String())
	}
	// Wait for resources to become ready by default. This matches common expectations
	// for "installed" status.
	args = append(args, "--wait")
	if chartRef.Kind == chartRefKindRepository {
		args = append(args, "--repo", chartRef.RepoURL)
		if chartRef.Version != "" {
			args = append(args, "--version", chartRef.Version)
		}
		if chartRef.Username != "" {
			args = append(args, "--username", chartRef.Username)
		}
		if chartRef.Password != "" {
			args = append(args, "--password", chartRef.Password)
		}
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

type chartRefKind string

const (
	chartRefKindRepository chartRefKind = "repository"
	chartRefKindLocal      chartRefKind = "local"
)

type chartRef struct {
	Kind chartRefKind

	// Ref is either the chart name (repo mode) or the local chart path.
	Ref string

	// repository-specific.
	RepoURL  string
	Version  string
	Username string
	Password string
}

func (c *CLIClient) resolveChartRef(ctx context.Context, chart steerv1alpha1.ChartSpec) (chartRef, func(), error) {
	cs := chart
	source := strings.TrimSpace(string(cs.Source))
	if source == "" {
		source = "repository"
	}

	switch source {
	case "repository":
		repo := cs.Repository
		if repo == nil {
			return chartRef{}, nil, fmt.Errorf("chart.repository is required")
		}
		if strings.TrimSpace(repo.Name) == "" {
			return chartRef{}, nil, fmt.Errorf("chart.repository.name is required")
		}
		if strings.TrimSpace(repo.URL) == "" {
			return chartRef{}, nil, fmt.Errorf("chart.repository.url is required")
		}

		// Special case: allow Git repos via repository.url=git+https://...@ref,
		// with chart.repository.name treated as the chart path inside the repo.
		if isGitHTTPSURL(repo.URL) {
			u, ref, err := parseGitHTTPSURL(repo.URL)
			if err != nil {
				return chartRef{}, nil, err
			}
			localPath, cleanup, err := c.cloneGitChart(ctx, u, ref, repo.Name, repo.Username, repo.Password)
			if err != nil {
				if cleanup != nil {
					cleanup()
				}
				return chartRef{}, nil, err
			}
			return chartRef{Kind: chartRefKindLocal, Ref: localPath}, cleanup, nil
		}

		return chartRef{
			Kind:     chartRefKindRepository,
			Ref:      repo.Name,
			RepoURL:  repo.URL,
			Version:  repo.Version,
			Username: repo.Username,
			Password: repo.Password,
		}, nil, nil
	case "local":
		l := cs.Local
		if l == nil || strings.TrimSpace(l.Path) == "" {
			return chartRef{}, nil, fmt.Errorf("chart.local.path is required")
		}
		return chartRef{Kind: chartRefKindLocal, Ref: l.Path}, nil, nil
	case "git":
		g := cs.Git
		if g == nil {
			return chartRef{}, nil, fmt.Errorf("chart.git is required")
		}
		if strings.TrimSpace(g.URL) == "" {
			return chartRef{}, nil, fmt.Errorf("chart.git.url is required")
		}
		if strings.TrimSpace(g.Path) == "" {
			return chartRef{}, nil, fmt.Errorf("chart.git.path is required")
		}
		u, err := url.Parse(g.URL)
		if err != nil {
			return chartRef{}, nil, fmt.Errorf("invalid chart.git.url: %w", err)
		}
		localPath, cleanup, err := c.cloneGitChart(ctx, u, strings.TrimSpace(g.Ref), g.Path, g.Username, g.Password)
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			return chartRef{}, nil, err
		}
		return chartRef{Kind: chartRefKindLocal, Ref: localPath}, cleanup, nil
	default:
		return chartRef{}, nil, fmt.Errorf("unsupported chart.source %q for CLI client", source)
	}
}

func isGitHTTPSURL(raw string) bool {
	return strings.HasPrefix(raw, "git+https://")
}

func parseGitHTTPSURL(raw string) (*url.URL, string, error) {
	// Expected format: git+https://example.com/org/repo.git@ref
	// Ref is optional (defaults to empty = default branch).
	if !strings.HasPrefix(raw, "git+https://") {
		return nil, "", fmt.Errorf("invalid git URL (expected git+https://): %q", raw)
	}
	trimmed := strings.TrimPrefix(raw, "git+")

	// Split ref by last '@' to avoid clobbering credentials (even though we don't
	// currently support credentials in URL).
	base := trimmed
	ref := ""
	if i := strings.LastIndex(trimmed, "@"); i > len("https://") {
		base = trimmed[:i]
		ref = trimmed[i+1:]
	}

	u, err := url.Parse(base)
	if err != nil {
		return nil, "", fmt.Errorf("invalid git URL %q: %w", raw, err)
	}
	if u.Scheme != "https" {
		return nil, "", fmt.Errorf("unsupported git URL scheme %q (must be https)", u.Scheme)
	}
	if u.Host == "" {
		return nil, "", fmt.Errorf("invalid git URL host: %q", raw)
	}
	return u, ref, nil
}

func (c *CLIClient) cloneGitChart(ctx context.Context, repoURL *url.URL, ref, chartPath, username, password string) (string, func(), error) {
	// Clone the repo into a temp dir, checkout ref, and return the absolute chart directory.
	// We use a temp dir to avoid accidental reuse across reconciles with different credentials.
	workDir, err := os.MkdirTemp("", "steer-git-*")
	if err != nil {
		return "", nil, err
	}
	cleanupWorkDir := func() { _ = os.RemoveAll(workDir) }

	// Never allow interactive prompts. If credentials are provided, use GIT_ASKPASS
	// so we don't embed secrets in command arguments.
	baseEnv := []string{"GIT_TERMINAL_PROMPT=0"}
	askpassEnv, cleanupAskPass, err := writeGitAskPass(workDir, username, password)
	if err != nil {
		cleanupWorkDir()
		return "", nil, err
	}
	baseEnv = append(baseEnv, askpassEnv...)

	cleanup := func() {
		if cleanupAskPass != nil {
			cleanupAskPass()
		}
		cleanupWorkDir()
	}

	remote := *repoURL
	// Explicitly drop any embedded credentials from URL (we only support providing
	// creds via username/password fields).
	remote.User = nil

	if strings.TrimSpace(ref) == "" {
		if _, err := runCmd(ctx, "git", []string{"clone", "--depth", "1", remote.String(), workDir}, baseEnv); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		// Prefer branch/tag clone first.
		if _, err := runCmd(ctx, "git", []string{"clone", "--depth", "1", "--branch", ref, "--single-branch", remote.String(), workDir}, baseEnv); err != nil {
			// Fallback: treat ref as commit-ish and fetch it explicitly.
			if _, err2 := runCmd(ctx, "git", []string{"clone", "--depth", "1", "--no-checkout", remote.String(), workDir}, baseEnv); err2 != nil {
				cleanup()
				return "", nil, fmt.Errorf("git clone failed: %w", err2)
			}
			if _, err2 := runCmd(ctx, "git", []string{"-C", workDir, "fetch", "--depth", "1", "origin", ref}, baseEnv); err2 != nil {
				cleanup()
				return "", nil, fmt.Errorf("git fetch failed: %w", err2)
			}
			if _, err2 := runCmd(ctx, "git", []string{"-C", workDir, "checkout", "--detach", "FETCH_HEAD"}, baseEnv); err2 != nil {
				cleanup()
				return "", nil, fmt.Errorf("git checkout failed: %w", err2)
			}
		}
	}

	chartDir := filepath.Join(workDir, filepath.Clean(chartPath))
	// Ensure chartDir is inside workDir.
	if !strings.HasPrefix(chartDir+string(filepath.Separator), workDir+string(filepath.Separator)) {
		cleanup()
		return "", nil, fmt.Errorf("invalid chart path %q (must be within repo)", chartPath)
	}
	if _, err := os.Stat(chartDir); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("chart path %q not found in repo: %w", chartPath, err)
	}
	// Basic sanity check.
	if _, err := os.Stat(filepath.Join(chartDir, "Chart.yaml")); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("chart path %q missing Chart.yaml: %w", chartPath, err)
	}

	return chartDir, cleanup, nil
}

func writeGitAskPass(workDir, username, password string) ([]string, func(), error) {
	if username == "" && password == "" {
		return nil, nil, nil
	}
	// The script itself contains no secrets; credentials are passed via env vars.
	path := filepath.Join(workDir, "steer-git-askpass.sh")
	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  *[Uu]sername*)\n" +
		"    printf '%s' \"$STEER_GIT_USERNAME\"\n" +
		"    ;;\n" +
		"  *)\n" +
		"    printf '%s' \"$STEER_GIT_PASSWORD\"\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.Remove(path) }
	return []string{
		"GIT_ASKPASS=" + path,
		"STEER_GIT_USERNAME=" + username,
		"STEER_GIT_PASSWORD=" + password,
	}, cleanup, nil
}

func runCmd(ctx context.Context, bin string, args []string, extraEnv []string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), extraEnv...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("%s %s failed: %w\n%s", bin, strings.Join(redactArgs(args), " "), err, buf.String())
	}
	return buf.String(), nil
}

var sensitiveArgKeys = []*regexp.Regexp{
	regexp.MustCompile(`^--password$`),
}

func redactArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		out = append(out, a)
		for _, re := range sensitiveArgKeys {
			if re.MatchString(a) {
				// redact next value if present.
				if i+1 < len(args) {
					i++
					out = append(out, "***")
				}
				break
			}
		}
	}
	return out
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
		return buf.String(), fmt.Errorf("helm %s failed: %w\n%s", strings.Join(redactArgs(args), " "), err, buf.String())
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
