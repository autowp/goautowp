package schema

import "github.com/doug-martin/goqu/v9"

const (
	VotingTableName = "voting"
)

var (
	VotingTable      = goqu.T(VotingTableName)
	VotingTableIDCol = VotingTable.Col("id")
)
