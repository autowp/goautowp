package goautowp

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TextGRPCServer struct {
	UnimplementedTextServer
	db *goqu.Database
}

func NewTextGRPCServer(
	db *goqu.Database,
) *TextGRPCServer {
	return &TextGRPCServer{
		db: db,
	}
}

func (s *TextGRPCServer) GetText(ctx context.Context, in *APIGetTextRequest) (*APIGetTextResponse, error) {
	var (
		lastRevision    int64
		currentRevision = in.Revision
		currentText     string
		currentUserID   int64
		prevRevision    int64
		prevText        string
		prevUserID      int64
		nextRevision    int64
	)

	err := s.db.QueryRowContext(
		ctx,
		"SELECT revision FROM textstorage_text WHERE id = ?",
		in.Id,
	).Scan(&lastRevision)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "NotFound")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if currentRevision == 0 {
		currentRevision = lastRevision
	}

	err = s.db.QueryRowContext(
		ctx,
		"SELECT text, user_id FROM textstorage_revision WHERE text_id = ? AND revision = ?",
		in.Id, currentRevision,
	).Scan(&currentText, &currentUserID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if currentRevision-1 > 0 {
		prevRevision = currentRevision - 1

		err = s.db.QueryRowContext(
			ctx,
			"SELECT text, user_id FROM textstorage_revision WHERE text_id = ? AND revision = ?",
			in.Id, prevRevision,
		).Scan(&prevText, &prevUserID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if currentRevision+1 <= lastRevision {
		nextRevision = currentRevision + 1
	}

	return &APIGetTextResponse{
		Current: &TextRevision{
			Text:     currentText,
			Revision: currentRevision,
			UserId:   currentUserID,
		},
		Prev: &TextRevision{
			Text:     prevText,
			Revision: prevRevision,
			UserId:   prevUserID,
		},
		Next: &TextRevision{
			Revision: nextRevision,
		},
	}, nil
}
