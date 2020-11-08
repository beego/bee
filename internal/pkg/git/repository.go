package git

import (
	"errors"
	"fmt"
	"github.com/beego/bee/internal/pkg/command"
	"github.com/beego/bee/internal/pkg/utils"
	beeLogger "github.com/beego/bee/logger"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// git tag
func GetTags(repoPath string, limit int) ([]string, error) {
	repo, err := OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}
	err = repo.Pull()
	if err != nil {
		return nil, err
	}
	list, err := repo.GetTags()
	if err != nil {
		return nil, err
	}
	if len(list) > limit {
		list = list[0:limit]
	}
	return list, nil
}

// clone repo
func CloneRepo(url string, dst string) (err error) {
	if utils.IsExist(dst) {
		return errors.New("dst is not empty, dst is " + dst)
	}
	if !utils.Mkdir(dst) {
		err = errors.New("make dir error, dst is " + dst)
		return
	}

	beeLogger.Log.Info("start git clone from " + url + ", to dst at " + dst)
	_, stderr, err := command.ExecCmd("git", "clone", url, dst)

	if err != nil {
		beeLogger.Log.Error("error git clone from " + url + ", to dst at " + dst)
		return concatenateError(err, stderr)
	}
	return nil
}

// CloneORPullRepo
func CloneORPullRepo(url string, dst string) error {
	if !utils.IsDir(dst) {
		return CloneRepo(url, dst)
	} else {
		utils.Mkdir(dst)

		repo, err := OpenRepository(dst)
		if err != nil {
			return err
		}

		return repo.Pull()
	}
}

//
func CloneRepoBranch(branch string, url string, dst string) error {
	_, stderr, err := command.ExecCmd("git", "clone", "-b", branch, url, dst)
	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}

// SortTag ...
type SortTag struct {
	data []string
}

// Len ...
func (t *SortTag) Len() int {
	return len(t.data)
}

// Swap ...
func (t *SortTag) Swap(i, j int) {
	t.data[i], t.data[j] = t.data[j], t.data[i]
}

// Less ...
func (t *SortTag) Less(i, j int) bool {
	return command.VerCompare(t.data[i], t.data[j]) == 1
}

// Sort ...
func (t *SortTag) Sort() []string {
	sort.Sort(t)
	return t.data
}

// Repository ...
type Repository struct {
	Path string
}

// OpenRepository ...
func OpenRepository(repoPath string) (*Repository, error) {
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	} else if !utils.IsDir(repoPath) {
		return nil, errors.New("no such file or directory")
	}

	return &Repository{Path: repoPath}, nil
}

// 拉取代码
func (repo *Repository) Pull() error {
	beeLogger.Log.Info("git pull " + repo.Path)
	_, stderr, err := command.ExecCmdDir(repo.Path, "git", "pull")
	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}

// 获取tag列表
func (repo *Repository) GetTags() ([]string, error) {
	stdout, stderr, err := command.ExecCmdDir(repo.Path, "git", "tag", "-l")
	if err != nil {
		return nil, concatenateError(err, stderr)
	}
	tags := strings.Split(stdout, "\n")
	tags = tags[:len(tags)-1]

	so := &SortTag{data: tags}
	return so.Sort(), nil
}

// 获取两个版本之间的修改日志
func (repo *Repository) GetChangeLogs(startVer, endVer string) ([]string, error) {
	// git log --pretty=format:"%cd %cn: %s" --date=iso v1.8.0...v1.9.0
	stdout, stderr, err := command.ExecCmdDir(repo.Path, "git", "log", "--pretty=format:%cd %cn: %s", "--date=iso", startVer+"..."+endVer)
	if err != nil {
		return nil, concatenateError(err, stderr)
	}

	logs := strings.Split(stdout, "\n")
	return logs, nil
}

// 获取两个版本之间的差异文件列表
func (repo *Repository) GetChangeFiles(startVer, endVer string, onlyFile bool) ([]string, error) {
	// git diff --name-status -b v1.8.0 v1.9.0
	param := "--name-status"
	if onlyFile {
		param = "--name-only"
	}
	stdout, stderr, err := command.ExecCmdDir(repo.Path, "git", "diff", param, "-b", startVer, endVer)
	if err != nil {
		return nil, concatenateError(err, stderr)
	}
	lines := strings.Split(stdout, "\n")
	return lines[:len(lines)-1], nil
}

// 获取两个版本间的新增或修改的文件数量
func (repo *Repository) GetDiffFileCount(startVer, endVer string) (int, error) {
	cmd := "git diff --name-status -b " + startVer + " " + endVer + " |grep -v ^D |wc -l"
	stdout, stderr, err := command.ExecCmdDir(repo.Path, "/bin/bash", "-c", cmd)
	if err != nil {
		return 0, concatenateError(err, stderr)
	}
	count, _ := strconv.Atoi(strings.TrimSpace(stdout))
	return count, nil
}

// 导出版本到tar包
func (repo *Repository) Export(startVer, endVer string, filename string) error {
	// git archive --format=tar.gz $endVer $(git diff --name-status -b $beginVer $endVer |grep -v ^D |grep -v Upgrade/ |awk '{print $2}') -o $tmpFile

	cmd := ""
	if startVer == "" {
		cmd = "git archive --format=tar " + endVer + " | gzip > " + filename
	} else {
		cmd = "git archive --format=tar " + endVer + " $(dgit diff --name-status -b " + startVer + " " + endVer + "|grep -v ^D |awk '{print $2}') | gzip > " + filename
	}

	_, stderr, err := command.ExecCmdDir(repo.Path, "/bin/bash", "-c", cmd)

	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}

func concatenateError(err error, stderr string) error {
	if len(stderr) == 0 {
		return err
	}
	return fmt.Errorf("%v: %s", err, stderr)
}
