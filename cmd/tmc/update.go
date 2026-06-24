package tmc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

const (
	defaultUpdateRegistryURL   = "https://registry.npmjs.org/@tmcopilot%2fcli"
	defaultUpdateCheckInterval = 2 * time.Hour
	automaticUpdateTimeout     = 1500 * time.Millisecond
	automaticInstallTimeout    = 5 * time.Minute
)

type updateCheckResult struct {
	CurrentVersion       string `json:"current_version"`
	LatestVersion        string `json:"latest_version,omitempty"`
	UpdateAvailable      bool   `json:"update_available"`
	UpdateCheckSupported bool   `json:"update_check_supported"`
	Channel              string `json:"channel,omitempty"`
	Source               string `json:"source,omitempty"`
	CheckedAt            string `json:"checked_at,omitempty"`
	InstallCommand       string `json:"install_command,omitempty"`
	ReleaseURL           string `json:"release_url,omitempty"`
	CachePath            string `json:"cache_path,omitempty"`
	InstallAttempted     bool   `json:"install_attempted,omitempty"`
	Installed            bool   `json:"installed,omitempty"`
	InstallError         string `json:"install_error,omitempty"`
	Message              string `json:"message,omitempty"`
}

type updateCheckCache struct {
	CheckedAt            string `json:"checked_at"`
	CurrentVersion       string `json:"current_version,omitempty"`
	LatestVersion        string `json:"latest_version,omitempty"`
	UpdateAvailable      bool   `json:"update_available,omitempty"`
	UpdateCheckSupported bool   `json:"update_check_supported,omitempty"`
	Channel              string `json:"channel,omitempty"`
	Source               string `json:"source,omitempty"`
	InstallCommand       string `json:"install_command,omitempty"`
	ReleaseURL           string `json:"release_url,omitempty"`
	InstallAttempted     bool   `json:"install_attempted,omitempty"`
	Installed            bool   `json:"installed,omitempty"`
	InstalledAt          string `json:"installed_at,omitempty"`
	InstallError         string `json:"install_error,omitempty"`
	Error                string `json:"error,omitempty"`
}

type npmPackageMetadata struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]json.RawMessage `json:"versions"`
}

type updateCheckOptions struct {
	CurrentVersion string
	RegistryURL    string
	Timeout        time.Duration
	Now            time.Time
}

var runUpdateInstaller = runUpdateInstallerCommand

func newUpdateCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update TMCopilot CLI",
		Long: `Check whether a newer TMCopilot CLI release is available and
install it when possible.

Automatic checks run at most once every two hours in interactive terminals. When a
newer release is available, the CLI runs the npm installer automatically.
Installer output is written to stderr so command stdout remains machine-readable.`,
		Example: `  tmc update
  tmc update check
  npx --yes @tmcopilot/cli@latest update
  npx --yes @tmcopilot/cli@experimental update`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateCommand(cmd, opts, true)
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check whether a newer CLI version is available",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateCommand(cmd, opts, false)
		},
	})
	return cmd
}

func runUpdateCommand(cmd *cobra.Command, opts *globalOptions, install bool) error {
	return handleCommand(cmd, func() error {
		rt, err := commandRuntime(cmd, opts, false)
		if err != nil {
			return err
		}
		result, err := checkForCLIUpdate(cmd.Context(), updateCheckOptions{
			CurrentVersion: version.Version,
			RegistryURL:    updateRegistryURL(),
			Timeout:        manualUpdateTimeout(opts),
			Now:            time.Now(),
		})
		if err != nil {
			return err
		}
		result.CachePath = mustUpdateCheckCachePath()
		if install && result.UpdateAvailable {
			installCtx, installCancel := context.WithTimeout(cmd.Context(), automaticInstallTimeout)
			defer installCancel()
			installErr := installCLIUpdate(installCtx, cmd, &result)
			result.InstallAttempted = true
			if installErr != nil {
				result.InstallError = installErr.Error()
			} else {
				result.Installed = true
			}
		}
		if err := saveUpdateCheckCacheFromResult(result, ""); err != nil {
			return err
		}
		return writeResult(rt, result, nil)
	})
}

func maybeRunLightweightAutomaticUpdateCheck(cmd *cobra.Command, args []string) *updateCheckResult {
	if cmd == nil || shouldSkipLightweightAutomaticUpdateCheck(cmd, args) {
		return nil
	}
	result, ok := runAutomaticUpdateProbe(firstCommandArg(args) == "version")
	if !ok || !result.UpdateAvailable {
		return nil
	}
	return &result
}

func maybeRunAutomaticUpdateCheck(cmd *cobra.Command) {
	if cmd == nil || shouldSkipAutomaticUpdateCheck(cmd) {
		return
	}
	result, ok := runAutomaticUpdateProbe(cmd.CommandPath() == "tmc version")
	if !ok || !result.UpdateAvailable {
		return
	}
	if automaticUpdateInstallDisabled() {
		writeUpdateAvailableNotice(cmd.ErrOrStderr(), result)
		return
	}
	installCtx, installCancel := context.WithTimeout(context.Background(), automaticInstallTimeout)
	defer installCancel()
	installErr := installCLIUpdate(installCtx, cmd, &result)
	result.InstallAttempted = true
	if installErr != nil {
		result.InstallError = installErr.Error()
		fmt.Fprintf(cmd.ErrOrStderr(), "Automatic update failed: %v\n", installErr)
		fmt.Fprintf(cmd.ErrOrStderr(), "Run manually: %s\n", result.InstallCommand)
	} else {
		result.Installed = true
	}
	_ = saveUpdateCheckCacheFromResult(result, "")
}

func runAutomaticUpdateProbe(force bool) (updateCheckResult, bool) {
	cache, _ := loadUpdateCheckCache()
	now := time.Now()
	if !force && !updateCheckDue(cache, now, updateCheckInterval()) {
		return updateCheckResult{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), automaticUpdateTimeout)
	defer cancel()
	result, err := checkForCLIUpdate(ctx, updateCheckOptions{
		CurrentVersion: version.Version,
		RegistryURL:    updateRegistryURL(),
		Timeout:        automaticUpdateTimeout,
		Now:            now,
	})
	if err != nil {
		_ = saveUpdateCheckCache(updateCheckCache{
			CheckedAt:      now.UTC().Format(time.RFC3339),
			CurrentVersion: version.Version,
			Source:         updateRegistryURL(),
			Error:          err.Error(),
		})
		return updateCheckResult{}, false
	}
	_ = saveUpdateCheckCacheFromResult(result, "")
	return result, true
}

func shouldSkipAutomaticUpdateCheck(cmd *cobra.Command) bool {
	if automaticUpdateCheckDisabled() || !releasedVersion(version.Version) {
		return true
	}
	if cmd != nil {
		path := cmd.CommandPath()
		if path == "tmc update" || strings.HasPrefix(path, "tmc update ") {
			return true
		}
		if path == "tmc uninstall" {
			return true
		}
		if flag := cmd.Root().PersistentFlags().Lookup("dry-run"); flag != nil && flag.Value.String() == "true" {
			return true
		}
	}
	return !isInteractiveStderr(cmd)
}

func shouldSkipLightweightAutomaticUpdateCheck(cmd *cobra.Command, args []string) bool {
	if automaticUpdateCheckDisabled() || !releasedVersion(version.Version) {
		return true
	}
	if isInteractiveStderr(cmd) {
		return true
	}
	switch firstCommandArg(args) {
	case "update", "uninstall":
		return true
	}
	return boolFlagEnabledInArgs(args, "dry-run")
}

func isInteractiveStderr(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	file, ok := cmd.ErrOrStderr().(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func writeUpdateAvailableNotice(w io.Writer, result updateCheckResult) {
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "\nUpdate available: tmc %s -> %s\n", result.CurrentVersion, result.LatestVersion)
	fmt.Fprintf(w, "Run: %s\n", result.InstallCommand)
}

func firstCommandArg(args []string) string {
	valueFlags := map[string]bool{
		"endpoint":        true,
		"format":          true,
		"idempotency-key": true,
		"output":          true,
		"profile":         true,
		"request-out":     true,
		"timeout":         true,
		"workspace":       true,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return ""
		}
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
		name, hasValue := longFlagNameAndValue(arg)
		if valueFlags[name] && !hasValue {
			i++
		}
	}
	return ""
}

func boolFlagEnabledInArgs(args []string, flagName string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		name, hasValue := longFlagNameAndValue(arg)
		if name != flagName {
			continue
		}
		if !hasValue {
			return true
		}
		value := strings.TrimSpace(strings.TrimPrefix(arg, "--"+flagName+"="))
		switch strings.ToLower(value) {
		case "", "1", "t", "true", "y", "yes", "on":
			return true
		default:
			return false
		}
	}
	return false
}

func longFlagNameAndValue(arg string) (string, bool) {
	if !strings.HasPrefix(arg, "--") || arg == "--" {
		return "", false
	}
	name := strings.TrimPrefix(arg, "--")
	name, _, hasValue := strings.Cut(name, "=")
	return name, hasValue
}

func checkForCLIUpdate(ctx context.Context, opts updateCheckOptions) (updateCheckResult, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	currentRaw := strings.TrimSpace(opts.CurrentVersion)
	current, ok := parseCLIVersion(currentRaw)
	if !ok {
		return updateCheckResult{
			CurrentVersion:       currentRaw,
			UpdateCheckSupported: false,
			UpdateAvailable:      false,
			CheckedAt:            now.UTC().Format(time.RFC3339),
			Message:              "update checks require a released CLI version",
		}, nil
	}
	metadata, err := fetchNpmPackageMetadata(ctx, opts.RegistryURL, opts.Timeout)
	if err != nil {
		return updateCheckResult{}, err
	}
	channel := updateChannelForVersion(current)
	latest, ok := latestVersionForChannel(metadata, channel)
	if !ok {
		return updateCheckResult{}, fmt.Errorf("no %s versions found in npm package metadata", channel)
	}
	updateAvailable := compareCLIVersion(latest, current) > 0
	latestDisplay := displayVersionLike(currentRaw, latest.Original)
	result := updateCheckResult{
		CurrentVersion:       currentRaw,
		LatestVersion:        latestDisplay,
		UpdateAvailable:      updateAvailable,
		UpdateCheckSupported: true,
		Channel:              channel,
		Source:               opts.RegistryURL,
		CheckedAt:            now.UTC().Format(time.RFC3339),
		InstallCommand:       installCommandForUpdateChannel(channel),
		ReleaseURL:           releaseURLForVersion(latest.Original),
	}
	if !updateAvailable {
		result.Message = "tmc is up to date"
	}
	return result, nil
}

func fetchNpmPackageMetadata(ctx context.Context, registryURL string, timeout time.Duration) (npmPackageMetadata, error) {
	if strings.TrimSpace(registryURL) == "" {
		registryURL = defaultUpdateRegistryURL
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL, nil)
	if err != nil {
		return npmPackageMetadata{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tmcopilot-cli/"+version.Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return npmPackageMetadata{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return npmPackageMetadata{}, fmt.Errorf("npm registry returned HTTP %d", resp.StatusCode)
	}
	var metadata npmPackageMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return npmPackageMetadata{}, fmt.Errorf("decode npm package metadata: %w", err)
	}
	if len(metadata.Versions) == 0 {
		return npmPackageMetadata{}, fmt.Errorf("npm package metadata did not include versions")
	}
	return metadata, nil
}

func updateRegistryURL() string {
	if value := strings.TrimSpace(os.Getenv("TMCOPILOT_UPDATE_REGISTRY_URL")); value != "" {
		return value
	}
	return defaultUpdateRegistryURL
}

func manualUpdateTimeout(opts *globalOptions) time.Duration {
	if opts != nil && opts.timeout > 0 && opts.timeout < 10*time.Second {
		return opts.timeout
	}
	return 10 * time.Second
}

func automaticUpdateCheckDisabled() bool {
	for _, key := range []string{"TMCOPILOT_NO_UPDATE_CHECK", "TMC_NO_UPDATE_CHECK"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		if value != "" && value != "0" && value != "false" && value != "no" {
			return true
		}
	}
	return false
}

func automaticUpdateInstallDisabled() bool {
	for _, key := range []string{"TMCOPILOT_NO_AUTO_UPDATE", "TMC_NO_AUTO_UPDATE"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		if value != "" && value != "0" && value != "false" && value != "no" {
			return true
		}
	}
	return false
}

func updateCheckInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("TMCOPILOT_UPDATE_CHECK_INTERVAL"))
	if value == "" {
		return defaultUpdateCheckInterval
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval < 0 {
		return defaultUpdateCheckInterval
	}
	return interval
}

func updateCheckDue(cache updateCheckCache, now time.Time, interval time.Duration) bool {
	if interval <= 0 || strings.TrimSpace(cache.CheckedAt) == "" {
		return true
	}
	checkedAt, err := time.Parse(time.RFC3339, cache.CheckedAt)
	if err != nil {
		return true
	}
	return now.Sub(checkedAt) >= interval
}

func updateCheckCachePath() (string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "update-check.json"), nil
}

func mustUpdateCheckCachePath() string {
	path, err := updateCheckCachePath()
	if err != nil {
		return ""
	}
	return path
}

func loadUpdateCheckCache() (updateCheckCache, error) {
	path, err := updateCheckCachePath()
	if err != nil {
		return updateCheckCache{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return updateCheckCache{}, nil
	}
	if err != nil {
		return updateCheckCache{}, err
	}
	var cache updateCheckCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		return updateCheckCache{}, err
	}
	return cache, nil
}

func saveUpdateCheckCacheFromResult(result updateCheckResult, lastError string) error {
	return saveUpdateCheckCache(updateCheckCache{
		CheckedAt:            result.CheckedAt,
		CurrentVersion:       result.CurrentVersion,
		LatestVersion:        result.LatestVersion,
		UpdateAvailable:      result.UpdateAvailable,
		UpdateCheckSupported: result.UpdateCheckSupported,
		Channel:              result.Channel,
		Source:               result.Source,
		InstallCommand:       result.InstallCommand,
		ReleaseURL:           result.ReleaseURL,
		InstallAttempted:     result.InstallAttempted,
		Installed:            result.Installed,
		InstalledAt:          installedAtForResult(result),
		InstallError:         result.InstallError,
		Error:                lastError,
	})
}

func saveUpdateCheckCache(cache updateCheckCache) error {
	if strings.TrimSpace(cache.CheckedAt) == "" {
		cache.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	}
	path, err := updateCheckCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func installedAtForResult(result updateCheckResult) string {
	if !result.Installed {
		return ""
	}
	if strings.TrimSpace(result.CheckedAt) != "" {
		return result.CheckedAt
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func updateChannelForVersion(v cliVersion) string {
	if v.PreReleaseID() == "experimental" {
		return "experimental"
	}
	return "latest"
}

func latestVersionForChannel(metadata npmPackageMetadata, channel string) (cliVersion, bool) {
	var best cliVersion
	var found bool
	for value := range metadata.Versions {
		candidate, ok := parseCLIVersion(value)
		if !ok {
			continue
		}
		if channel == "latest" && candidate.Prerelease != "" {
			continue
		}
		if channel != "latest" && candidate.PreReleaseID() != channel {
			continue
		}
		if !found || compareCLIVersion(candidate, best) > 0 {
			best = candidate
			found = true
		}
	}
	if found {
		return best, true
	}
	if tagged, ok := metadata.DistTags[channel]; ok {
		return parseCLIVersion(tagged)
	}
	return cliVersion{}, false
}

func installCommandForUpdateChannel(channel string) string {
	return strings.Join(installCommandArgsForUpdateChannel(channel), " ")
}

func installCommandArgsForUpdateChannel(channel string) []string {
	if channel == "" {
		channel = "latest"
	}
	return []string{"npx", "--yes", "@tmcopilot/cli@" + channel, "update"}
}

func installCLIUpdate(ctx context.Context, cmd *cobra.Command, result *updateCheckResult) error {
	if result == nil || strings.TrimSpace(result.Channel) == "" {
		return fmt.Errorf("update channel is required")
	}
	args := installCommandArgsForUpdateChannel(result.Channel)
	if len(args) == 0 {
		return fmt.Errorf("update install command is empty")
	}
	result.InstallCommand = strings.Join(args, " ")
	stderr := cmd.ErrOrStderr()
	fmt.Fprintf(stderr, "\nUpdate available: tmc %s -> %s\n", result.CurrentVersion, result.LatestVersion)
	fmt.Fprintf(stderr, "Installing: %s\n", result.InstallCommand)
	if err := runUpdateInstaller(ctx, args, cmd.InOrStdin(), stderr, stderr); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "TMCopilot CLI updated to %s.\n", result.LatestVersion)
	return nil
}

func runUpdateInstallerCommand(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("update install command is empty")
	}
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func releaseURLForVersion(value string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(value), "v")
	if normalized == "" {
		return ""
	}
	return "https://github.com/huski-inc/tmcopilot-cli/releases/tag/v" + normalized
}

func displayVersionLike(currentRaw string, latest string) string {
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if strings.HasPrefix(strings.TrimSpace(currentRaw), "v") {
		return "v" + latest
	}
	return latest
}

func releasedVersion(value string) bool {
	_, ok := parseCLIVersion(value)
	return ok
}

type cliVersion struct {
	Original   string
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

func parseCLIVersion(value string) (cliVersion, bool) {
	original := strings.TrimSpace(value)
	normalized := strings.TrimPrefix(original, "v")
	if normalized == "" {
		return cliVersion{}, false
	}
	mainPart, prerelease, _ := strings.Cut(normalized, "-")
	parts := strings.Split(mainPart, ".")
	if len(parts) != 3 {
		return cliVersion{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return cliVersion{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return cliVersion{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return cliVersion{}, false
	}
	return cliVersion{
		Original:   normalized,
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, true
}

func (v cliVersion) PreReleaseID() string {
	if strings.TrimSpace(v.Prerelease) == "" {
		return ""
	}
	id, _, _ := strings.Cut(v.Prerelease, ".")
	return id
}

func compareCLIVersion(a, b cliVersion) int {
	for _, pair := range [][2]int{
		{a.Major, b.Major},
		{a.Minor, b.Minor},
		{a.Patch, b.Patch},
	} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}
	return comparePrerelease(a.Prerelease, b.Prerelease)
}

func comparePrerelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1
		}
		if i >= len(bParts) {
			return 1
		}
		cmp := comparePrereleaseIdentifier(aParts[i], bParts[i])
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

func comparePrereleaseIdentifier(a, b string) int {
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)
	aIsNum := aErr == nil
	bIsNum := bErr == nil
	switch {
	case aIsNum && bIsNum:
		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
		return 0
	case aIsNum:
		return -1
	case bIsNum:
		return 1
	default:
		return strings.Compare(a, b)
	}
}
