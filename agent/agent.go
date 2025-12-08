package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/salzr/acert/proto/agentservice/v1"
)

// TODO: Create options for the agent client
// The client will be charged with ensuring that the issued certificate is valid. If the certificate is not valid, the client will attempt to renew it.
func Run(ctx context.Context) error {
	log := ctx.Value("logger").(*zap.Logger)
	log = log.With(zap.String("service", "agent"))

	// TODO: The keypairs would be pulled by config
	cert, err := tls.LoadX509KeyPair(
		"/Users/dsalazar/Documents/workspace/github.com/salzr/acert/config/certmanager/agent.crt",
		"/Users/dsalazar/Documents/workspace/github.com/salzr/acert/config/certmanager/agent.key")
	if err != nil {
		log.Fatal("failed to load client cert", zap.Error(err))
	}

	ca := x509.NewCertPool()
	caFilePath := "/Users/dsalazar/Documents/workspace/github.com/salzr/acert/config/certmanager/server-ca.crt"
	caBytes, err := os.ReadFile(caFilePath)
	if err != nil {
		log.Fatal("failed to read ca cert", zap.Error(err))
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		log.Fatal("failed to parse", zap.String("filepath", caFilePath))
	}

	tlsConfig := &tls.Config{
		ServerName:   "server.acert.salzr.localhost",
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
	}

	conn, err := grpc.NewClient("server.acert.salzr.localhost:50051", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	stream, err := client.Poll(ctx)
	if err != nil {
		log.Fatal("Failed to poll", zap.Error(err))
	}

	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				log.Fatal("Failed to receive", zap.Error(err))
			}
			if task := res.GetServerTask(); task != nil {
				log.Info("Received task", zap.String("taskId", task.TaskId), zap.String("command", task.Command))
			} else if status := res.GetServerStatus(); status != nil {
				log.Info("Received status", zap.String("message", status.Message))
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

OUTER:
	for {
		select {
		case <-ticker.C:
			heartbeat := &pb.AgentRequest{
				AgentId: "agent-1",
				Payload: &pb.AgentRequest_Heartbeat{
					Heartbeat: &pb.AgentHeartbeat{
						Timestamp: time.Now().Unix(),
					},
				},
			}
			if err := stream.Send(heartbeat); err != nil {
				log.Fatal("Failed to send heartbeat", zap.Error(err))
			}
		case <-done:
			stream.CloseSend()
			break OUTER
		}
	}

	return nil
}
