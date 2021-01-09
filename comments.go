package goautowp

import (
	"database/sql"
	"github.com/autowp/goautowp/util"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Comments service
type Comments struct {
	db            *sql.DB
	userExtractor *UserExtractor
}

// APIUser APIUser
type APIUser struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Deleted  bool     `json:"deleted"`
	LongAway bool     `json:"long_away"`
	Green    bool     `json:"green"`
	Route    []string `json:"route"`
	Identity *string  `json:"identity"`
}

// DBUser DBUser
type DBUser struct {
	ID         int
	Name       string
	Deleted    bool
	Identity   *string
	LastOnline *time.Time
	Role       string
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

// Routes adds routes
func (s *Comments) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/comment/votes", func(c *gin.Context) {

		idStr := c.Query("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		votes, err := s.getVotes(id)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if votes == nil {
			c.Status(http.StatusNotFound)
			return
		}

		positive := make([]*APIUser, 0)
		for _, user := range votes.PositiveVotes {
			extracted, err := s.userExtractor.Extract(&user)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			positive = append(positive, extracted)
		}

		negative := make([]*APIUser, 0)
		for _, user := range votes.NegativeVotes {
			extracted, err := s.userExtractor.Extract(&user)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			negative = append(negative, extracted)
		}

		c.JSON(http.StatusOK, gin.H{
			"positive": positive,
			"negative": negative,
		})
	})
}
