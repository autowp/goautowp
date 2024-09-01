package schema

import (
	"github.com/doug-martin/goqu/v9"
)

const (
	PicturesModerVotesTableName             = "pictures_moder_votes"
	PicturesModerVotesTableUserIDColName    = "user_id"
	PicturesModerVotesTablePictureIDColName = "picture_id"
	PicturesModerVotesTableVoteColName      = "vote"
	PicturesModerVotesTableReasonColName    = "reason"
	PicturesModerVotesTableDayDateColName   = "day_date"
)

var (
	PicturesModerVotesTable             = goqu.T(PicturesModerVotesTableName)
	PicturesModerVotesTableUserIDCol    = PicturesModerVotesTable.Col(PicturesModerVotesTableUserIDColName)
	PicturesModerVotesTablePictureIDCol = PicturesModerVotesTable.Col(PicturesModerVotesTablePictureIDColName)
	PicturesModerVotesTableVoteCol      = PicturesModerVotesTable.Col(PicturesModerVotesTableVoteColName)
	PicturesModerVotesTableReasonCol    = PicturesModerVotesTable.Col(PicturesModerVotesTableReasonColName)
)

type PictureModerVoteRow struct {
	UserID int64  `db:"user_id"`
	Reason string `db:"reason"`
}
