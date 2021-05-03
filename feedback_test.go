package goautowp

import (
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/peer"
	"net"
	"testing"
)

func TestFeedbackNoBody(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{})
	require.Error(t, err)
}

func TestFeedbackEmptyValues(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{
		Name:    "",
		Email:   "",
		Message: "",
	})
	require.Error(t, err)
}

func TestFeedbackEmptyName(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{
		Name:    "",
		Email:   "test@example.com",
		Message: "message",
	})
	require.Error(t, err)
}

func TestFeedbackEmptyEmail(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{
		Name:    "",
		Email:   "",
		Message: "message",
	})
	require.Error(t, err)
}

func TestFeedbackEmptyMessage(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{
		Name:    "user",
		Email:   "test@example.com",
		Message: "",
	})
	require.Error(t, err)
}

func TestFeedbackMessage(t *testing.T) {
	config := LoadConfig()
	config.Feedback.Captcha = false
	srv, err := NewContainer(config).GetGRPCServer()
	require.NoError(t, err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.IPv4(192, 168, 0, 1)}})

	_, err = srv.CreateFeedback(ctx, &APICreateFeedbackRequest{
		Name:    "user",
		Email:   "test@example.com",
		Message: "message",
	})
	require.NoError(t, err)
}
