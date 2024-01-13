package prenv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mumoshu/prenv/cmd"
	"github.com/mumoshu/prenv/envvar"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"
)

func TestDispatchThenGitOps(t *testing.T) {
	hooks := testServerRepoHooks{
		repos: map[string]*testServerHooks{},
	}

	var (
		sourceRepo = "mumoshu/prenv-source"
		targetRepo = "mumoshu/prenv-target"

		testdataDir           = "gitops"
		testdataSourceRepoDir = filepath.Join(testdataDir, "repositories", "mumoshu", "prenv-source")
		testdataTargetRepoDir = filepath.Join(testdataDir, "repositories", "mumoshu", "prenv-target")
	)

	ts, err := newTestServer([]string{
		sourceRepo,
		targetRepo,
	}, &hooks)
	require.NoError(t, err)

	baseDir := t.TempDir()

	gitServerRoot := filepath.Join(baseDir, "gitserver")

	gts, err := newTestGitServer(gitServerRoot, os.Getenv(envvar.GitHubToken), testdataDir, []string{
		sourceRepo,
		targetRepo,
	})
	require.NoError(t, err)

	gtsURL := strings.Replace(gts.URL+"/", "127.0.0.1", "localhost", 1)

	sourceRepoDir := createDirFromTestdataDir(t, baseDir, testdataSourceRepoDir)
	targetRepoDir := createDirFromTestdataDir(t, baseDir, testdataTargetRepoDir)

	wd, err := os.Getwd()
	require.NoError(t, err)

	err = run(args{
		Command: []string{"apply"},
		Env: map[string]string{
			// BaseURL must have a trailing slash, as required by go-github
			envvar.GitHubBaseURL:       ts.URL + "/",
			envvar.GitHubEventPath:     filepath.Join(wd, "testdata", testdataDir, "events", "01-pull_request.json"),
			envvar.GitHubRepository:    sourceRepo,
			envvar.GitHubEnterpriseURL: gtsURL,
		},
		Dir: sourceRepoDir,
	})
	require.NoError(t, err)

	rawConfig := "dedicated:\n  components:\n    sourceapp:\n      render:\n        git:\n          repo: mumoshu/prenv-source\n          branch: main\n          path: deploy\n          push: true\n        files:\n        - name: kubernetes/test.configmap.yaml\n          contentTemplate: |\n            apiVersion: v1\n            kind: ConfigMap\n            metadata:\n              name: test\n            data:\n              pr_nums.json: |\n                {{ .PullRequest.Numbers | toJson }}\n        - name: terraform/test.auto.tfvars.json\n          contentTemplate: |\n            {\"prenv_pull_request_numbers\": {{ .PullRequest.Numbers | toJson }}}\n    targetapp:\n      render:\n        git:\n          repo: mumoshu/prenv-target\n          branch: main\n          path: apps\n          push: true\n        pullRequest: {}\n        repositoryDispatch:\n          owner: mumoshu\n          repo: prenv-target\n        files:\n        - nameTemplate: app.{{ .PullRequest.Number }}.yaml\n          contentTemplate: |\n            kind: Application\n            apiVersion: argoproj.io/v1alpha1\n            metadata:\n              name: app-{{ .PullRequest.Number }}\n            spec:\n              project: default\n              source:\n                repoURL: https://github.com/mumoshu/prenv-target\n                targetRevision: main\n                path: kustomize\n              destination:\n                server: https://kubernetes.default.svc\n                namespace: default\n              kustomize:\n                namePrefix: app-{{ .PullRequest.Number }}-\n                images:\n                - name: myapp\n                  newTag: {{ .PullRequest.HeadSHA }}\n              syncPolicy:\n                automated:\n                  prune: true\n                  selfHeal: true\n                  allowEmpty: true\n                  apply:\n                    force: true\n              syncWave: 1\n              syncOptions:\n              - CreateNamespace=true\nargs:\n  name: prenv-123\n  appnametemplate: '{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}-{{\n    .ShortName }}'\n  pullRequest:\n    number: 123\n    repository: mumoshu/prenv-source\n"

	wantRepositoryDispatches := []repositoryDispatch{
		{
			Event: "prenv-apply",
			ClientPayload: map[string]interface{}{
				"raw_config":   rawConfig,
				"triggered_by": []interface{}{"pr-targetapp-render"},
			},
		},
	}

	require.Equal(t, wantRepositoryDispatches, hooks.repos[targetRepo].RepositoryDispatches)
	require.Empty(t, hooks.repos[targetRepo].PullRequests)

	githubEventPath := filepath.Join(baseDir, "events", "repository_dispatch.json")
	eventDir := filepath.Dir(githubEventPath)
	require.NoError(t, os.MkdirAll(eventDir, 0755))

	eventData, err := json.Marshal(hooks.repos[targetRepo].RepositoryDispatches[0].ToActionEvent())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(githubEventPath, eventData, 0644))

	err = run(args{
		Command: []string{"action"},
		Env: map[string]string{
			envvar.GitHubEventPath: githubEventPath,
			// BaseURL must have a trailing slash, as required by go-github
			envvar.GitHubBaseURL:       ts.URL + "/",
			envvar.GitHubRepository:    targetRepo,
			envvar.GitHubEnterpriseURL: gtsURL,
		},
		Dir: targetRepoDir,
	})
	require.NoError(t, err)

	wantPullRequests := []pullRequest{
		{
			Title: "automated commit",
			// Head:  "pr-123",
			Base: "refs/heads/main",
			Body: "n/a",
		},
	}
	require.Equal(t, wantPullRequests, hooks.repos[targetRepo].PullRequests)
}

func createDirFromTestdataDir(t *testing.T, baseDir, testdataDir string) string {
	t.Helper()

	sourceRepoDir := filepath.Join(baseDir, testdataDir)
	templateDir := filepath.Join("testdata", testdataDir)

	require.NoError(t, os.MkdirAll(sourceRepoDir, 0755))

	// Walk the template directory and create the same directory structure in the source repository directory
	// and copy the files from the template directory to the source repository directory.
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(filepath.Join(sourceRepoDir, rel), 0755)
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(filepath.Join(sourceRepoDir, rel), b, 0644)
	})
	require.NoError(t, err)
	return sourceRepoDir
}

type pullRequest struct {
	Title string `json:"title"`
	// Head  string `json:"head"`
	Base string `json:"base"`
	Body string `json:"body"`
}

type repositoryDispatch struct {
	Event         string                 `json:"event_type"`
	ClientPayload map[string]interface{} `json:"client_payload"`
}

func (r *repositoryDispatch) ToActionEvent() repositoryDispatchActionEvent {
	return repositoryDispatchActionEvent{
		Action:        r.Event,
		ClientPayload: r.ClientPayload,
	}
}

type repositoryDispatchActionEvent struct {
	Action        string                 `json:"action"`
	ClientPayload map[string]interface{} `json:"client_payload"`
}

type args struct {
	Command []string
	Env     map[string]string
	Dir     string
}

var (
	initOSArgs   []string
	dir          string
	preservedEnv []string
)

func init() {
	var err error
	dir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	preservedEnv = os.Environ()
	initOSArgs = append([]string{}, os.Args...)
}

func run(args args) error {
	defer func() {
		for k := range args.Env {
			_ = os.Unsetenv(k)
		}

		_ = os.Chdir(dir)
	}()

	for k, v := range args.Env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}

	if err := os.Chdir(args.Dir); err != nil {
		return err
	}

	os.Args = nil
	os.Args = append([]string{}, "prenv")
	os.Args = append(os.Args, args.Command...)

	return cmd.Main()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		dstPath := strings.Replace(path, src, dst, 1)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, b, 0644)
	})
}

// gitServerRoot is the directory that contains the git repositories.
// token supposed to be the GitHub API token used to authenticate the git requests.
// repos is the list of repositories that the git server serves,
// in the form of "owner/repo".
func newTestGitServer(gitServerRoot, token string, testdataDir string, repos []string) (*httptest.Server, error) {
	ownerRepos := map[string][]string{}

	for _, repo := range repos {
		split := strings.Split(repo, "/")
		owner, name := split[0], split[1]

		ownerRepos[owner] = append(ownerRepos[owner], name)
	}

	mux := http.NewServeMux()

	for owner, repos := range ownerRepos {
		ownerRoot, err := filepath.Abs(filepath.Join(gitServerRoot, owner))
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(ownerRoot, 0755); err != nil {
			return nil, err
		}

		for _, repo := range repos {
			repoRoot := filepath.Join(ownerRoot, repo) + ".git"

			gitInitBareCmd := exec.Command("git", "init", "--bare", repoRoot)

			r, err := gitInitBareCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git init --bare: %w: %s", err, r)
			}

			repoWorktreeRoot := filepath.Join(ownerRoot, repo)

			gitCloneCmd := exec.Command("git", "clone", repoRoot, repoWorktreeRoot)

			r, err = gitCloneCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git clone: %w: %s", err, r)
			}

			srcDir := filepath.Join("testdata", testdataDir, "repositories", owner, repo)
			if err := copyDir(srcDir, repoWorktreeRoot); err != nil {
				return nil, err
			}

			gitAddCmd := exec.Command("git", "add", ".")
			gitAddCmd.Dir = repoWorktreeRoot

			r, err = gitAddCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git add: %w: %s", err, r)
			}

			gitCommitCmd := exec.Command("git", "commit", "-m", "initial commit")
			gitCommitCmd.Dir = repoWorktreeRoot

			r, err = gitCommitCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git commit: %w: %s", err, r)
			}

			gitPushCmd := exec.Command("git", "push", "origin", "master:main")
			gitPushCmd.Dir = repoWorktreeRoot

			r, err = gitPushCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git push: %w: %s", err, r)
			}

			// See https://stackoverflow.com/a/15631690 why we need to change the HEAD to main
			// Note that this works only after we created the main branch on the remote
			//
			// Without this, `git clone` still tries to checkout the master branch,
			// which doesn't exist yet on the remote as we pushed only the main branch.
			gitChangeHeadCmd := exec.Command("git", "symbolic-ref", "HEAD", "refs/heads/main")
			gitChangeHeadCmd.Dir = repoRoot

			r, err = gitChangeHeadCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git symbolic-ref HEAD refs/heads/main: %w: %s", err, r)
			}
		}
	}

	g := gitkit.New(gitkit.Config{
		Dir:  gitServerRoot,
		Auth: true,
	})

	g.AuthFunc = func(cred gitkit.Credential, req *gitkit.Request) (bool, error) {
		return cred.Password == token, nil
	}

	// gitkit supports namespaces so you don't need multiple servers
	// to serve owner/repo1 and owner/repo2.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		g.ServeHTTP(w, r)
	})

	return httptest.NewServer(mux), nil
}

func newTestServer(repos []string, hooks *testServerRepoHooks) (*httptest.Server, error) {
	mux := http.NewServeMux()

	for _, repo := range repos {
		h := &testServerHooks{}
		hooks.repos[repo] = h

		mux.HandleFunc(fmt.Sprintf("/repos/%s/dispatches", repo), func(w http.ResponseWriter, r *http.Request) {
			var req repositoryDispatch
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			h.RepositoryDispatches = append(h.RepositoryDispatches, req)

			w.WriteHeader(http.StatusAccepted)
		})

		mux.HandleFunc(fmt.Sprintf("/repos/%s/pulls", repo), func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(h.PullRequests)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				return
			case http.MethodPost:
				var req pullRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				h.PullRequests = append(h.PullRequests, req)

				w.WriteHeader(http.StatusCreated)
			}
		})
	}

	return httptest.NewServer(mux), nil
}

type testServerRepoHooks struct {
	repos map[string]*testServerHooks
}

type testServerHooks struct {
	RepositoryDispatches []repositoryDispatch
	PullRequests         []pullRequest
}
