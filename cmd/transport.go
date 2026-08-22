package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kana/storage"
)

// RemoteLocation represents a parsed remote target (local path or SSH).
type RemoteLocation struct {
	IsSSH   bool
	Host    string // e.g. "user@10.18.0.97" or "host"
	Port    string // e.g. "22" or custom
	Path    string // e.g. "/home/user/project" or "C:\dev\project"
}

// ParseRemoteLocation parses a remote string into a RemoteLocation.
func ParseRemoteLocation(raw string) (*RemoteLocation, error) {
	raw = strings.TrimSpace(raw)

	// Case 1: ssh://user@host:port/path
	if strings.HasPrefix(raw, "ssh://") {
		trimmed := strings.TrimPrefix(raw, "ssh://")
		parts := strings.SplitN(trimmed, "/", 2)
		hostPart := parts[0]
		pathPart := "/"
		if len(parts) > 1 {
			pathPart = "/" + parts[1]
		}
		port := "22"
		if strings.Contains(hostPart, ":") {
			hp := strings.SplitN(hostPart, ":", 2)
			hostPart = hp[0]
			port = hp[1]
		}
		return &RemoteLocation{
			IsSSH: true,
			Host:  hostPart,
			Port:  port,
			Path:  pathPart,
		}, nil
	}

	// Case 2: user@host:/path/to/repo or host:/path/to/repo (SCP syntax)
	if strings.Contains(raw, ":") && !strings.Contains(raw, `:\`) && !strings.HasPrefix(raw, ".") {
		parts := strings.SplitN(raw, ":", 2)
		// Check if first part looks like a Windows drive letter (e.g. C:)
		if len(parts[0]) == 1 && (parts[0][0] >= 'a' && parts[0][0] <= 'z' || parts[0][0] >= 'A' && parts[0][0] <= 'Z') {
			// Windows drive letter -> local path
			abs, err := filepath.Abs(raw)
			if err != nil {
				return nil, err
			}
			return &RemoteLocation{IsSSH: false, Path: abs}, nil
		}
		return &RemoteLocation{
			IsSSH: true,
			Host:  parts[0],
			Port:  "22",
			Path:  parts[1],
		}, nil
	}

	// Case 3: Local filesystem path
	abs, err := filepath.Abs(raw)
	if err != nil {
		return nil, err
	}
	return &RemoteLocation{
		IsSSH: false,
		Path:  abs,
	}, nil
}

// ResolveRemoteEndpoint resolves a remote name (from store) or raw URL into a RemoteLocation.
func ResolveRemoteEndpoint(store *storage.Storage, remoteOrURL string) (*RemoteLocation, error) {
	url, err := store.GetRemote(remoteOrURL)
	if err == nil && url != "" {
		return ParseRemoteLocation(url)
	}
	return ParseRemoteLocation(remoteOrURL)
}

// GetRemoteStreamHead fetches the latest snapshot hash from a remote location.
func GetRemoteStreamHead(loc *RemoteLocation, stream string) (string, error) {
	if !loc.IsSSH {
		remoteStore, err := storage.OpenRepo(loc.Path)
		if err != nil {
			return "", fmt.Errorf("failed to open local remote at %s: %w", loc.Path, err)
		}
		defer remoteStore.Close()
		return remoteStore.GetStreamHead(stream)
	}

	// Over SSH
	sshArgs := []string{}
	if loc.Port != "" && loc.Port != "22" {
		sshArgs = append(sshArgs, "-p", loc.Port)
	}
	cmdStr := fmt.Sprintf("kana internal stream-head %q %q", loc.Path, stream)
	sshArgs = append(sshArgs, loc.Host, cmdStr)

	cmd := exec.Command("ssh", sshArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ssh connection failed to %s: %w", loc.Host, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// PushBundleToRemote pushes a sync bundle to a remote location.
func PushBundleToRemote(loc *RemoteLocation, bundle *storage.SyncBundle) error {
	bundleBytes, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("failed to encode bundle: %w", err)
	}

	if !loc.IsSSH {
		remoteStore, err := storage.OpenRepo(loc.Path)
		if err != nil {
			return fmt.Errorf("failed to open local remote at %s: %w", loc.Path, err)
		}
		defer remoteStore.Close()
		return remoteStore.ImportSyncBundle(bundle)
	}

	// Pipe to remote over SSH
	sshArgs := []string{}
	if loc.Port != "" && loc.Port != "22" {
		sshArgs = append(sshArgs, "-p", loc.Port)
	}
	cmdStr := fmt.Sprintf("kana internal import-bundle %q", loc.Path)
	sshArgs = append(sshArgs, loc.Host, cmdStr)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = bytes.NewReader(bundleBytes)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remote import failed over ssh to %s: %w", loc.Host, err)
	}

	return nil
}

// FetchBundleFromRemote pulls a sync bundle from a remote location.
func FetchBundleFromRemote(loc *RemoteLocation, stream, sinceHash string) (*storage.SyncBundle, error) {
	if !loc.IsSSH {
		remoteStore, err := storage.OpenRepo(loc.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to open local remote at %s: %w", loc.Path, err)
		}
		defer remoteStore.Close()
		return remoteStore.ExportSyncBundle(stream, sinceHash)
	}

	// Pull from remote over SSH
	sshArgs := []string{}
	if loc.Port != "" && loc.Port != "22" {
		sshArgs = append(sshArgs, "-p", loc.Port)
	}
	cmdStr := fmt.Sprintf("kana internal export-bundle %q %q %q", loc.Path, stream, sinceHash)
	sshArgs = append(sshArgs, loc.Host, cmdStr)

	cmd := exec.Command("ssh", sshArgs...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ssh export failed from %s: %w", loc.Host, err)
	}

	var bundle storage.SyncBundle
	if err := json.Unmarshal(out, &bundle); err != nil {
		return nil, fmt.Errorf("invalid bundle response from remote: %w", err)
	}

	return &bundle, nil
}
