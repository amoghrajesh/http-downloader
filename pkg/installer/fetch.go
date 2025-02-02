package installer

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/linuxsuren/http-downloader/pkg/common"
	"github.com/mitchellh/go-homedir"
	"io"
	"os"
	"path"
	"strings"
)

const (
	// ConfigGitHub is the default git repository URI
	ConfigGitHub = "https://github.com/LinuxSuRen/hd-home"
	// ConfigBranch is the default branch name of hd-home git repository
	ConfigBranch = "master"
)

var configRepos = map[string]string{
	"github": ConfigGitHub,
	"gitee":  "https://gitee.com/LinuxSuRen/hd-home",
}

// GetConfigDir returns the directory of the config
func GetConfigDir() (configDir string, err error) {
	var userHome string
	if userHome, err = homedir.Dir(); err == nil {
		configDir = path.Join(userHome, "/.config/hd-home")
	}
	return
}

// FetchLatestRepo fetches the hd-home as the config
func FetchLatestRepo(provider string, branch string, progress io.Writer) (err error) {
	repoAddr, ok := configRepos[provider]
	if !ok {
		repoAddr = ConfigGitHub
	}

	if branch == "" {
		branch = ConfigBranch
	}

	remoteName := "origin"
	if repoAddr != ConfigGitHub {
		remoteName = provider
	}

	var configDir string
	if configDir, err = GetConfigDir(); err != nil {
		return
	}

	if ok, _ := common.PathExists(configDir); ok {
		var repo *git.Repository
		if repo, err = git.PlainOpen(configDir); err == nil {
			var wd *git.Worktree

			if wd, err = repo.Worktree(); err == nil {
				if err = makeSureRemove(remoteName, repoAddr, repo); err != nil {
					err = fmt.Errorf("cannot add remote: %s, address: %s, error: %v", remoteName, repoAddr, err)
					return
				}

				if err = wd.Pull(&git.PullOptions{
					RemoteName: remoteName,
					Progress:   progress,
					Force:      true,
				}); err != nil && err != git.NoErrAlreadyUpToDate {
					err = fmt.Errorf("failed to pull git repository '%s', error: %v", repo, err)
					return
				}
				err = nil
			}
		} else {
			err = fmt.Errorf("failed to open git local repository, error: %v", err)
		}
	} else {
		if _, err = git.PlainClone(configDir, false, &git.CloneOptions{
			RemoteName: remoteName,
			URL:        repoAddr,
			Progress:   progress,
		}); err != nil {
			err = fmt.Errorf("failed to clone git repository '%s' into '%s', error: %v", repoAddr, configDir, err)
		}
	}

	if err != nil && strings.Contains(err.Error(), "exit status 128") {
		// target directory was created accidentally, remove it then try again
		_ = os.RemoveAll(configDir)
		return FetchLatestRepo(repoAddr, branch, progress)
	}
	return
}

func makeSureRemove(name, repoAddr string, repo *git.Repository) (err error) {
	if _, err = repo.Remote(name); err != nil {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: name,
			URLs: []string{repoAddr},
		})
	}
	return
}
