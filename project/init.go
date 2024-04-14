package project

import (
	"fmt"
	"github.com/alibaba/git-repo-go/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alibaba/git-repo-go/file"
	"github.com/alibaba/git-repo-go/path"
)

// IsRepoInitialized indicates repository is initialized or not.
func (v Project) IsRepoInitialized() bool {
	if v.ObjectsGitDir != "" {
		if !path.IsGitDir(v.ObjectsGitDir) {
			return false
		}
	}
	if !path.IsGitDir(v.GitDir) {
		return false
	}
	return true
}

// GitInit starts to init repositories.
func (v *Project) GitInit() error {
	var (
		referenceGitDir string
		remoteURL       string
		err             error
	)

	remoteURL, err = v.GetRemoteURL()
	if err != nil {
		return err
	}

	objectsRepo := v.ObjectsRepository()
	if objectsRepo != nil && v.GitDir != objectsRepo.GitDir {
		objectsRepo.Init("", "", "") // init git under project-objects path
		objectsRepo.InitHooks()
		v.Repository.InitByLink(v.RemoteName, remoteURL, objectsRepo) // init git with link under projects path
	} else {
		v.Repository.Init(v.RemoteName, remoteURL, referenceGitDir)
	}

	// TODO: install hooks
	return nil
}

func (v *Repository) initMissing() error {
	var err error

	if _, err = os.Stat(v.GitDir); err != nil {
		return err
	}

	dirs := []string{
		"hooks",
		"branches",
		"hooks",
		"info",
		"refs",
	}
	files := map[string]string{
		"description": fmt.Sprintf("Repository: %s, path: %s\n", v.Name, v.GitDir),
		"config":      "[core]\n\trepositoryformatversion = 0\n",
		"HEAD":        "ref: refs/heads/master\n",
	}

	for _, dir := range dirs {
		dir = filepath.Join(v.GitDir, dir)
		if _, err = os.Stat(dir); err == nil {
			continue
		}
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	for name, content := range files {
		name = filepath.Join(v.GitDir, name)
		if _, err = os.Stat(name); err == nil {
			continue
		}
		f, err := file.New(name).OpenCreateRewrite()
		if err != nil {
			return err
		}
		f.WriteString(content)
		f.Close()
	}

	if !v.IsBare {
		cfg := v.Config()
		cfg.Unset("core.bare")
		cfg.Set("core.logAllRefUpdates", "true")
		cfg.Save(v.configFile())
	}

	return nil
}

// Init runs git-init on repository.
func (v *Repository) Init(remoteName, remoteURL, referenceGitDir string) error {
	var err error

	cmdArgs := []string{
		GIT,
		"init",
		"-q",
		"--bare",
		v.GitDir,
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = nil
	err = cmd.Run()
	if err != nil {
		return err
	}

	if !v.IsBare {
		cfg := v.Config()
		cfg.Unset("core.bare")
		cfg.Set("core.logAllRefUpdates", "true")
		v.SaveConfig(cfg)
	}

	if remoteName != "" && remoteURL != "" {
		if !strings.HasSuffix(remoteURL, ".git") {
			remoteURL += ".git"
		}
		u := v.GitConfigRemoteURL(remoteName)
		if u != remoteURL {
			err = v.setRemote(remoteName, remoteURL)
			if err != nil {
				return err
			}
		}
	}

	if referenceGitDir != "" {
		v.setAlternates(referenceGitDir)
	}

	return nil
}

// InitByLink starts to init repository by attaching other repository.
func (v *Repository) InitByLink(remoteName, remoteURL string, repo *Repository) error {
	var err error

	if !repo.Exists() {
		return fmt.Errorf("attach a non-exist repo: %s", repo.GitDir)
	}
	repo.initMissing()

	if repo.GitDir == v.GitDir {
		return nil
	}

	err = os.MkdirAll(v.GitDir, 0755)
	if err != nil {
		return err
	}

	items := []string{
		"objects",
		"description",
		"info",
		"hooks",
		"svn",
		"rr-cache",
	}
	for _, item := range items {
		source := filepath.Join(repo.GitDir, item)
		target := filepath.Join(v.GitDir, item)
		if _, err = os.Stat(source); err != nil {
			continue
		}
		relpath, err := filepath.Rel(v.GitDir, source)
		if err != nil {
			relpath = source
		}
		err = os.Symlink(relpath, target)
		if err != nil {
			break
		}
	}

	// Create config file
	v.Init(remoteName, remoteURL, "")

	return nil
}

func (v *Repository) InitHooks() error {
	hookdir, err := config.GetRepoHooksDir()
	if err != nil {
		return fmt.Errorf("fail to get hook path %s", err)
	}

	for name, _ := range config.GerritHooks {
		target := filepath.Join(v.GitDir, "hooks", name)
		source := filepath.Join(hookdir, name)
		err = os.Symlink(source, target)
		if err != nil {
			return fmt.Errorf("fail to set link for hook %s: %s", name, err)
		}
	}
	return nil
}
