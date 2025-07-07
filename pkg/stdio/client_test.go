package stdio

import (
	"context"
	"time"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/mudler/LocalAGI/pkg/xlog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client *Client
	)

	BeforeEach(func() {
		client = NewClient(baseURL)
	})

	AfterEach(func() {
		if client != nil {
			Expect(client.Close()).To(Succeed())
		}
	})

	Context("Process Management", func() {
		It("should create and stop a process", func() {
			ctx := context.Background()
			// Use a command that doesn't exit immediately
			process, err := client.CreateProcess(ctx, "sh", []string{"-c", "echo 'Hello, World!'; sleep 10"}, []string{}, "test-group")
			Expect(err).NotTo(HaveOccurred())
			Expect(process).NotTo(BeNil())
			Expect(process.ID).NotTo(BeEmpty())

			// Get process IO
			reader, writer, err := client.GetProcessIO(process.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(reader).NotTo(BeNil())
			Expect(writer).NotTo(BeNil())

			// Write to process
			_, err = writer.Write([]byte("test input\n"))
			Expect(err).NotTo(HaveOccurred())

			// Read from process with timeout
			buf := make([]byte, 1024)
			readDone := make(chan struct{})
			var readErr error
			var readN int

			go func() {
				readN, readErr = reader.Read(buf)
				close(readDone)
			}()

			// Wait for read with timeout
			select {
			case <-readDone:
				Expect(readErr).NotTo(HaveOccurred())
				Expect(readN).To(BeNumerically(">", 0))
				Expect(string(buf[:readN])).To(ContainSubstring("Hello, World!"))
			case <-time.After(5 * time.Second):
				Fail("Timeout waiting for process output")
			}

			// Stop the process
			err = client.StopProcess(process.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should manage process groups", func() {
			ctx := context.Background()
			groupID := "test-group"

			// Create multiple processes in the same group
			process1, err := client.CreateProcess(ctx, "sh", []string{"-c", "echo 'Process 1'; sleep 1"}, []string{}, groupID)
			Expect(err).NotTo(HaveOccurred())
			Expect(process1).NotTo(BeNil())

			process2, err := client.CreateProcess(ctx, "sh", []string{"-c", "echo 'Process 2'; sleep 1"}, []string{}, groupID)
			Expect(err).NotTo(HaveOccurred())
			Expect(process2).NotTo(BeNil())

			// Get group processes
			processes, err := client.GetGroupProcesses(groupID)
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(2))

			// List groups
			groups := client.ListGroups()
			Expect(groups).To(ContainElement(groupID))

			// Stop the group
			err = client.StopGroup(groupID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should run a one-time process", func() {
			ctx := context.Background()
			output, err := client.RunProcess(ctx, "echo", []string{"One-time process"}, []string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("One-time process"))
		})

		It("should handle process with environment variables", func() {
			ctx := context.Background()
			env := []string{"TEST_VAR=test_value"}
			process, err := client.CreateProcess(ctx, "sh", []string{"-c", "env | grep TEST_VAR; sleep 1"}, env, "test-group")
			Expect(err).NotTo(HaveOccurred())
			Expect(process).NotTo(BeNil())

			// Get process IO
			reader, _, err := client.GetProcessIO(process.ID)
			Expect(err).NotTo(HaveOccurred())

			// Read environment variables with timeout
			buf := make([]byte, 1024)
			readDone := make(chan struct{})
			var readErr error
			var readN int

			go func() {
				readN, readErr = reader.Read(buf)
				close(readDone)
			}()

			// Wait for read with timeout
			select {
			case <-readDone:
				Expect(readErr).NotTo(HaveOccurred())
				Expect(readN).To(BeNumerically(">", 0))
				Expect(string(buf[:readN])).To(ContainSubstring("TEST_VAR=test_value"))
			case <-time.After(5 * time.Second):
				Fail("Timeout waiting for process output")
			}

			// Stop the process
			err = client.StopProcess(process.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle long-running processes", func() {
			ctx := context.Background()
			process, err := client.CreateProcess(ctx, "sh", []string{"-c", "echo 'Starting long process'; sleep 5"}, []string{}, "test-group")
			Expect(err).NotTo(HaveOccurred())
			Expect(process).NotTo(BeNil())

			// Get process IO
			reader, _, err := client.GetProcessIO(process.ID)
			Expect(err).NotTo(HaveOccurred())

			// Read initial output
			buf := make([]byte, 1024)
			readDone := make(chan struct{})
			var readErr error
			var readN int

			go func() {
				readN, readErr = reader.Read(buf)
				close(readDone)
			}()

			// Wait for read with timeout
			select {
			case <-readDone:
				Expect(readErr).NotTo(HaveOccurred())
				Expect(readN).To(BeNumerically(">", 0))
				Expect(string(buf[:readN])).To(ContainSubstring("Starting long process"))
			case <-time.After(5 * time.Second):
				Fail("Timeout waiting for process output")
			}

			// Wait a bit to ensure process is running
			time.Sleep(time.Second)

			// Stop the process
			err = client.StopProcess(process.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("MCP", func() {
			ctx := context.Background()
			process, err := client.CreateProcess(ctx,
				"docker", []string{"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"},
				[]string{"GITHUB_PERSONAL_ACCESS_TOKEN=test"}, "test-group")
			Expect(err).NotTo(HaveOccurred())
			Expect(process).NotTo(BeNil())
			Expect(process.ID).NotTo(BeEmpty())

			defer client.StopProcess(process.ID)

			// MCP client

			read, writer, err := client.GetProcessIO(process.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(read).NotTo(BeNil())
			Expect(writer).NotTo(BeNil())

			transport := stdio.NewStdioServerTransportWithIO(read, writer)

			// Create a new client
			mcpClient := mcp.NewClient(transport)
			// Initialize the client
			response, e := mcpClient.Initialize(ctx)
			Expect(e).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			Expect(mcpClient.Ping(ctx)).To(Succeed())

			xlog.Debug("Client initialized: %v", response.Instructions)

			alltools := []mcp.ToolRetType{}
			var cursor *string
			for {
				tools, err := mcpClient.ListTools(ctx, cursor)
				Expect(err).NotTo(HaveOccurred())
				Expect(tools).NotTo(BeNil())
				Expect(tools.Tools).NotTo(BeEmpty())
				alltools = append(alltools, tools.Tools...)

				if tools.NextCursor == nil {
					break // No more pages
				}
				cursor = tools.NextCursor
			}

			for _, tool := range alltools {
				xlog.Debug("Tool: %v", tool)
			}
		})
	})
})
