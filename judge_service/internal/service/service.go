package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	ty "github.com/DeadlyParkour777/code-checker/judge_service/internal/types"
	problempb "github.com/DeadlyParkour777/code-checker/pkg/problem"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LanguageConfig struct {
	Image          string
	CodeFileName   string
	DockerfileTmpl string
}

var languageConfigs = map[string]LanguageConfig{
	"go": {
		Image:        "golang:1.24-alpine",
		CodeFileName: "main.go",
		DockerfileTmpl: `
FROM %[1]s AS builder
WORKDIR /app
COPY . .
RUN go mod init sandbox && go mod tidy
RUN go build -o /app/main .

FROM scratch
WORKDIR /app
COPY --from=builder /app/main .
CMD ["/app/main"]
`,
	},
	"python": {
		Image:        "python:3.11-alpine",
		CodeFileName: "main.py",
		DockerfileTmpl: `
FROM %[1]s
WORKDIR /app
COPY . .
CMD ["python", "main.py"]
`,
	},
}

const sharedSubmissionsDir = "/tmp/submissions"

type Service interface {
	ProcessSubmission(ctx context.Context, submission *ty.SubmissionEvent)
}
type service struct {
	kafkaProducer *kafka.Writer
	dockerClient  *client.Client
	timeout       time.Duration
	hostTempPath  string
	problemClient problempb.ProblemServiceClient
}

func NewService(producer *kafka.Writer, timeout time.Duration, hostTempPath string, problemServiceAddr string) Service {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
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
		hostTempPath:  hostTempPath,
		problemClient: problemClient,
	}
}

func (s *service) ProcessSubmission(ctx context.Context, submission *ty.SubmissionEvent) {
	log.Printf("Started processing submission %s", submission.SubmissionID)
	go func() {
		initialTimeout := 10 * time.Second
		jobCtx, cancel := context.WithTimeout(context.Background(), initialTimeout)
		defer cancel()

		result, err := s.judge(jobCtx, submission)
		if err != nil {
			result = &ty.ResultEvent{
				SubmissionID: submission.SubmissionID,
				Status:       "RE",
				Message:      err.Error(),
			}
		}

		err = s.kafkaProducer.WriteMessages(jobCtx, kafka.Message{Value: result.Marshal()})
		if err != nil {
			log.Printf("Failed to write result for submission %s: %v", submission.SubmissionID, err)
		} else {
			log.Printf("Finished processing submission %s with status %s", submission.SubmissionID, result.Status)
		}
	}()
}

func (s *service) judge(ctx context.Context, submission *ty.SubmissionEvent) (*ty.ResultEvent, error) {
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

	tempDir, err := os.MkdirTemp(sharedSubmissionsDir, "sub-*")
	if err != nil {
	}
	defer os.RemoveAll(tempDir)

	dockerfileContent := fmt.Sprintf(langConfig.DockerfileTmpl, langConfig.Image)
	if err := os.WriteFile(filepath.Join(tempDir, "Dockerfile.sandbox"), []byte(dockerfileContent), 0644); err != nil {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "RE",
			Message:      "Failed to create sandbox Dockerfile",
		}, nil
	}
	if err := os.WriteFile(filepath.Join(tempDir, langConfig.CodeFileName), []byte(submission.Code), 0644); err != nil {
		return &ty.ResultEvent{SubmissionID: submission.SubmissionID,
			Status:  "RE",
			Message: "Failed to write code to file",
		}, nil
	}

	imageName := "sandbox-" + uuid.New().String()
	err = s.buildSandboxImage(ctx, tempDir, imageName)
	if err != nil {
		return &ty.ResultEvent{
			SubmissionID: submission.SubmissionID,
			Status:       "CE",
			Message:      fmt.Sprintf("Compilation Error: %v", err),
		}, nil
	}
	defer s.dockerClient.ImageRemove(ctx, imageName, image.RemoveOptions{Force: true})

	executionTimeout := (s.timeout + time.Second) * time.Duration(len(testCases))
	runCtx, runCancel := context.WithTimeout(ctx, executionTimeout)
	defer runCancel()

	for i, testCase := range testCases {
		log.Printf("Running test case %d for submission %s", i+1, submission.SubmissionID)

		internalTC := &ty.TestCase{
			Input:  testCase.GetInputData(),
			Output: testCase.GetOutputData(),
		}

		status, output, err := s.runTestCase(runCtx, imageName, internalTC)
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

func (s *service) buildSandboxImage(ctx context.Context, dir, imageName string) error {
	buildContext, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}
	defer buildContext.Close()

	resp, err := s.dockerClient.ImageBuild(ctx, buildContext, build.ImageBuildOptions{
		Dockerfile:  "Dockerfile.sandbox",
		Tags:        []string{imageName},
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("error reading build response: %w", err)
	}

	if strings.Contains(strings.ToLower(buf.String()), "error:") {
		return fmt.Errorf("build failed: %s", buf.String())
	}

	return nil
}

func (s *service) runTestCase(ctx context.Context, imageName string, tc *ty.TestCase) (string, string, error) {
	cont, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:        imageName,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
	}, &container.HostConfig{
		Resources: container.Resources{Memory: 128 * 1024 * 1024, NanoCPUs: int64(0.5 * 1e9)},
	}, nil, nil, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to create container: %w", err)
	}
	defer s.dockerClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true})

	hijackedResp, err := s.dockerClient.ContainerAttach(ctx, cont.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to attach to container: %w", err)
	}
	defer hijackedResp.Close()

	if err := s.dockerClient.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
		return "", "", fmt.Errorf("failed to start container: %w", err)
	}

	if _, err := hijackedResp.Conn.Write([]byte(tc.Input)); err != nil {
		return "", "", fmt.Errorf("failed to write to stdin: %w", err)
	}

	hijackedResp.CloseWrite()

	resultC, errC := s.dockerClient.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	ctxTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	select {
	case <-ctxTimeout.Done():
		return "TLE", "Time Limit Exceeded", nil
	case err := <-errC:
		return "", "", fmt.Errorf("error waiting for container: %w", err)
	case result := <-resultC:
		outputBuf := new(bytes.Buffer)
		io.Copy(outputBuf, hijackedResp.Reader)

		if result.StatusCode != 0 {
			return "RE", fmt.Sprintf("Runtime Error (Exit Code: %d)\n%s", result.StatusCode, outputBuf.String()), nil
		}
		programOutput := ""
		if outputBuf.Len() > 8 {
			programOutput = outputBuf.String()[8:]
		} else {
			programOutput = outputBuf.String()
		}

		if result.StatusCode != 0 {
			return "RE", fmt.Sprintf("Runtime Error (Exit Code: %d)\n%s", result.StatusCode, programOutput), nil
		}

		if strings.TrimSpace(programOutput) != strings.TrimSpace(tc.Output) {
			return "WA", fmt.Sprintf("Wrong Answer.\nExpected:\n%s\nGot:\n%s", tc.Output, programOutput), nil
		}
	}
	return "AC", "", nil
}
