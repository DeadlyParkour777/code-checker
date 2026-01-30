package service

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	problempb "github.com/DeadlyParkour777/code-checker/pkg/problem"
	ty "github.com/DeadlyParkour777/code-checker/services/judge_service/internal/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LanguageConfig struct {
	CodeFileName string
}

var languageConfigs = map[string]LanguageConfig{
	"go": {
		CodeFileName: "main.go",
	},
	"python": {
		CodeFileName: "main.py",
	},
}

const (
	buildTimeout = 120 * time.Second
)

const runtimeImage = "code-checker-judge-runtime:latest"
const workVolume = "submissions-data"

//go:embed runtime/Dockerfile
var runtimeDockerfile []byte

//go:embed runtime/runner.sh
var runtimeRunnerScript []byte

type Service interface {
	ProcessSubmission(ctx context.Context, submission *ty.SubmissionEvent) error
}

type service struct {
	kafkaProducer *kafka.Writer
	dockerClient  *client.Client
	timeout       time.Duration
	workDir       string
	workerPool    chan string
	problemClient problempb.ProblemServiceClient
}

func NewService(
	producer *kafka.Writer,
	timeout time.Duration,
	workDir string,
	workerCount int,
	problemServiceAddr string,
) Service {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	if err := ensureRuntimeImage(dockerCli, runtimeImage); err != nil {
		log.Fatalf("Failed to ensure runtime image: %v", err)
	}

	if err := ensureWorkVolume(dockerCli, workVolume); err != nil {
		log.Fatalf("Failed to ensure work volume: %v", err)
	}

	workerPool := make(chan string, workerCount)
	if err := createWorkerContainers(dockerCli, runtimeImage, workVolume, workDir, workerCount, workerPool); err != nil {
		log.Fatalf("Failed to create worker containers: %v", err)
	}

	conn, err := grpc.NewClient(problemServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to problem service: %v", err)
	}
	problemClient := problempb.NewProblemServiceClient(conn)

	return &service{
		kafkaProducer: producer,
		dockerClient:  dockerCli,
		timeout:       timeout,
		workDir:       workDir,
		workerPool:    workerPool,
		problemClient: problemClient,
	}
}

func ensureRuntimeImage(dockerCli *client.Client, imageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, _, err := dockerCli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("image inspect failed: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "judge-runtime-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, "Dockerfile"), runtimeDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write runtime Dockerfile: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "runner.sh"), runtimeRunnerScript, 0755); err != nil {
		return fmt.Errorf("failed to write runtime runner: %w", err)
	}

	buildContext, err := archive.TarWithOptions(tempDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}
	defer buildContext.Close()

	resp, err := dockerCli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Dockerfile:  "Dockerfile",
		Tags:        []string{imageName},
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return fmt.Errorf("failed to build runtime image: %w", err)
	}
	defer resp.Body.Close()

	if err := parseBuildErrors(resp.Body); err != nil {
		return err
	}

	return nil
}

func ensureWorkVolume(dockerCli *client.Client, volumeName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := dockerCli.VolumeInspect(ctx, volumeName)
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("volume inspect failed: %w", err)
	}

	_, err = dockerCli.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
	if err != nil {
		return fmt.Errorf("volume create failed: %w", err)
	}
	return nil
}

func createWorkerContainers(
	dockerCli *client.Client,
	imageName string,
	volumeName string,
	workDir string,
	workerCount int,
	pool chan<- string,
) error {
	for i := 0; i < workerCount; i++ {
		name := fmt.Sprintf("judge-worker-%s-%d", uuid.New().String(), i)
		cont, err := dockerCli.ContainerCreate(context.Background(), &container.Config{
			Image: imageName,
			Cmd:   []string{"sh", "-c", "trap : TERM INT; sleep infinity & wait"},
		}, &container.HostConfig{
			Resources:   container.Resources{Memory: 128 * 1024 * 1024, NanoCPUs: int64(0.5 * 1e9)},
			NetworkMode: "none",
			Mounts: []mount.Mount{
				{Type: mount.TypeVolume, Source: volumeName, Target: workDir},
			},
		}, nil, nil, name)
		if err != nil {
			return fmt.Errorf("failed to create worker container: %w", err)
		}

		if err := dockerCli.ContainerStart(context.Background(), cont.ID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start worker container: %w", err)
		}

		pool <- cont.ID
	}
	return nil
}

func (s *service) ProcessSubmission(ctx context.Context, submission *ty.SubmissionEvent) error {
	log.Printf("Started processing submission %s", submission.SubmissionID)

	workerID := <-s.workerPool
	defer func() { s.workerPool <- workerID }()

	result, err := s.judge(ctx, submission, workerID)
	if err != nil {
		result = &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      err.Error(),
		}
	}

	err = s.kafkaProducer.WriteMessages(ctx, kafka.Message{Value: result.Marshal()})
	if err != nil {
		log.Printf("Failed to write result for submission %s: %v", submission.SubmissionID, err)
		return err
	}

	log.Printf("Finished processing submission %s with status %s", submission.SubmissionID, result.Status)
	return nil
}

func (s *service) judge(ctx context.Context, submission *ty.SubmissionEvent, workerID string) (*ty.ResultEvent, error) {
	langConfig, ok := languageConfigs[submission.Language]
	if !ok {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      "Unsupported language",
		}, nil
	}

	log.Printf("Fetching test cases for problem %s", submission.ProblemID)
	resp, err := s.problemClient.GetTestCases(ctx, &problempb.GetTestCasesRequest{ProblemId: submission.ProblemID})
	if err != nil {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      fmt.Sprintf("Failed to get test cases: %v", err),
		}, err
	}
	testCases := resp.GetTestCases()
	log.Printf("Received %d test cases", len(testCases))
	if len(testCases) == 0 {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "AC",
			Message:      "No test cases found for this problem.",
		}, nil
	}

	if err := os.MkdirAll(s.workDir, 0755); err != nil {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      "Failed to ensure work dir",
		}, err
	}

	subDir, err := os.MkdirTemp(s.workDir, "sub-*")
	if err != nil {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      "Failed to create temp dir",
		}, err
	}
	defer os.RemoveAll(subDir)

	if err := os.WriteFile(filepath.Join(subDir, langConfig.CodeFileName), []byte(submission.Code), 0644); err != nil {
		return &ty.ResultEvent{SubmissionID: submission.SubmissionID,
			Status:  "RE",
			Message: "Failed to write code to file",
		}, nil
	}

	binPath := filepath.Join(subDir, "app.bin")
	if submission.Language == "go" {
		buildCtx, cancelBuild := context.WithTimeout(ctx, buildTimeout+5*time.Second)
		defer cancelBuild()
		stdout, stderr, exitCode, err := s.execInWorker(buildCtx, workerID, []string{
			"judge-runner",
			"--phase", "compile",
			"--lang", submission.Language,
			"--workdir", subDir,
			"--outbin", binPath,
			"--timeout", fmt.Sprintf("%d", int(buildTimeout.Seconds())),
		}, "")
		if err != nil {
			return &ty.ResultEvent{
				SubmissionID: submission.SubmissionID,
				Status:       "CE",
				Message:      fmt.Sprintf("Compilation Error: %v", err),
			}, nil
		}
		if exitCode != 0 {
			msg := strings.TrimSpace(stderr)
			if msg == "" {
				msg = strings.TrimSpace(stdout)
			}
			return &ty.ResultEvent{
				SubmissionID: submission.SubmissionID,
				Status:       "CE",
				Message:      fmt.Sprintf("Compilation Error: %s", msg),
			}, nil
		}
	}

	for i, testCase := range testCases {
		log.Printf("Running test case %d for submission %s", i+1, submission.SubmissionID)

		internalTC := &ty.TestCase{
			Input:  testCase.GetInputData(),
			Output: testCase.GetOutputData(),
		}

		runCtx, cancelRun := context.WithTimeout(ctx, s.timeout+5*time.Second)
		status, output, err := s.runTestCase(runCtx, workerID, submission.Language, subDir, binPath, internalTC)
		cancelRun()
		if err != nil {
			return &ty.ResultEvent{
				SubmissionID: submission.SubmissionID,
				Status:       "RE",
				Message:      err.Error(),
			}, nil
		}
		if status != "AC" {
			return &ty.ResultEvent{
				SubmissionID: submission.SubmissionID,
				Status:       status,
				Message:      output,
			}, nil
		}
	}

	return &ty.ResultEvent{SubmissionID: submission.SubmissionID,
		Status:  "AC",
		Message: "All tests passed",
	}, nil
}

func (s *service) runTestCase(
	ctx context.Context,
	workerID string,
	lang string,
	workDir string,
	binPath string,
	tc *ty.TestCase,
) (string, string, error) {
	stdout, stderr, exitCode, err := s.execInWorker(ctx, workerID, []string{
		"judge-runner",
		"--phase", "run",
		"--lang", lang,
		"--workdir", workDir,
		"--outbin", binPath,
		"--timeout", fmt.Sprintf("%d", int(s.timeout.Seconds())),
	}, tc.Input)
	if err != nil {
		return "RE", err.Error(), nil
	}

	if exitCode == 124 || exitCode == 137 {
		return "TLE", "Time Limit Exceeded", nil
	}

	if exitCode != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = strings.TrimSpace(stdout)
		}
		return "RE", fmt.Sprintf("Runtime Error (Exit Code: %d)\n%s", exitCode, msg), nil
	}

	programOutput := stdout
	if strings.TrimSpace(programOutput) != strings.TrimSpace(tc.Output) {
		return "WA", fmt.Sprintf("Wrong Answer.\nExpected:\n%s\nGot:\n%s", tc.Output, programOutput), nil
	}

	return "AC", "", nil
}

func (s *service) execInWorker(
	ctx context.Context,
	workerID string,
	cmd []string,
	stdin string,
) (string, string, int, error) {
	execResp, err := s.dockerClient.ContainerExecCreate(ctx, workerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Cmd:          cmd,
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("exec create failed: %w", err)
	}

	attachResp, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", "", 0, fmt.Errorf("exec attach failed: %w", err)
	}
	defer attachResp.Close()

	if stdin != "" {
		if _, err := attachResp.Conn.Write([]byte(stdin)); err != nil {
			return "", "", 0, fmt.Errorf("failed to write to stdin: %w", err)
		}
	}
	attachResp.CloseWrite()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	done := make(chan struct{})
	go func() {
		_, _ = stdcopy.StdCopy(stdoutBuf, stderrBuf, attachResp.Reader)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return "", "", 0, ctx.Err()
	case <-done:
	}

	for {
		inspect, err := s.dockerClient.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return "", "", 0, fmt.Errorf("exec inspect failed: %w", err)
		}
		if !inspect.Running {
			return stdoutBuf.String(), stderrBuf.String(), inspect.ExitCode, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func parseBuildErrors(r io.Reader) error {
	dec := json.NewDecoder(r)
	var out bytes.Buffer
	for dec.More() {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			return fmt.Errorf("build output parse error: %w", err)
		}
		if msg.Stream != "" {
			if out.Len() < 64*1024 {
				_, _ = out.WriteString(msg.Stream)
			}
		}
		if msg.Error != "" {
			tail := strings.TrimSpace(out.String())
			if tail == "" {
				return fmt.Errorf("build failed: %s", msg.Error)
			}
			return fmt.Errorf("build failed: %s\n%s", msg.Error, tail)
		}
	}
	return nil
}
