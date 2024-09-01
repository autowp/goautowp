package schema

import "github.com/doug-martin/goqu/v9"

const (
	TelegramChatTableName = "telegram_chat"
)

var (
	TelegramChatTable            = goqu.T(TelegramChatTableName)
	TelegramChatTableChatIDCol   = TelegramChatTable.Col("chat_id")
	TelegramChatTableUserIDCol   = TelegramChatTable.Col("user_id")
	TelegramChatTableMessagesCol = TelegramChatTable.Col("messages")
)
