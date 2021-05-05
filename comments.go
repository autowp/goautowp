package goautowp

import (
	"database/sql"
	"github.com/autowp/goautowp/util"
)

type CommentsType int

const (
	CommentsTypePictureID    CommentsType = 1
	CommentsTypeItemID       CommentsType = 2
	CommentsTypeVotingID     CommentsType = 3
	CommentsTypeArticleID    CommentsType = 4
	CommentsTypeForumTopicID CommentsType = 5
)

// Comments service
type Comments struct {
	db            *sql.DB
	userExtractor *UserExtractor
}

type getVotesResult struct {
	PositiveVotes []DBUser
	NegativeVotes []DBUser
}

// NewComments constructor
func NewComments(db *sql.DB, userExtractor *UserExtractor) *Comments {

	return &Comments{
		db:            db,
		userExtractor: userExtractor,
	}
}

func (s *Comments) getVotes(id int) (*getVotesResult, error) {

	rows, err := s.db.Query(`
		SELECT users.id, users.name, users.deleted, users.identity, users.last_online, users.role, comment_vote.vote
		FROM comment_vote
			INNER JOIN users ON comment_vote.user_id = users.id
		WHERE comment_vote.comment_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	positiveVotes := make([]DBUser, 0)
	negativeVotes := make([]DBUser, 0)
	for rows.Next() {
		var r DBUser
		var vote int
		err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &vote)
		if err != nil {
			return nil, err
		}
		if vote > 0 {
			positiveVotes = append(positiveVotes, r)
		} else {
			negativeVotes = append(negativeVotes, r)
		}

	}

	return &getVotesResult{
		PositiveVotes: positiveVotes,
		NegativeVotes: negativeVotes,
	}, nil
}
