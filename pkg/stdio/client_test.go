package stdio

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client  *Client
		baseURL string
	)

	BeforeEach(func() {
		baseURL = os.Getenv("STDIO_SERVER_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
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
	})
})
