package schema

import "github.com/doug-martin/goqu/v9"

const (
	TableArticles            = "articles"
	TableAttrsAttributes     = "attrs_attributes"
	TableAttrsListOptions    = "attrs_list_options"
	TableAttrsTypes          = "attrs_types"
	TableAttrsUnits          = "attrs_units"
	TableAttrsUserValues     = "attrs_user_values"
	TableAttrsValues         = "attrs_values"
	TableAttrsZones          = "attrs_zones"
	TableAttrsZoneAttributes = "attrs_zone_attributes"
	TableCarTypes            = "car_types"

	CommentMessageTableName                      = "comment_message"
	CommentMessageTableModeratorAttentionColName = "moderator_attention"
	CommentMessageTableDeletedColName            = "deleted"
	CommentMessageTableDeleteDateColName         = "delete_date"
	CommentMessageTableRepliesCountColName       = "replies_count"

	CommentTopicTableName               = "comment_topic"
	CommentTopicSubscribeTableName      = "comment_topic_subscribe"
	TableCommentTopicView               = "comment_topic_view"
	CommentVoteTableName                = "comment_vote"
	TableContact                        = "contact"
	TableDfDistance                     = "df_distance"
	DfHashTableName                     = "df_hash"
	TableFormattedImage                 = "formated_image"
	ForumsThemesTableName               = "forums_themes"
	ForumsThemeParentTableName          = "forums_theme_parent"
	ForumsTopicsTableName               = "forums_topics"
	TableHtmls                          = "htmls"
	TableImage                          = "image"
	ItemTableName                       = "item"
	TableItemParentCache                = "item_parent_cache"
	TableItemLanguage                   = "item_language"
	TableItemParent                     = "item_parent"
	TableItemParentLanguage             = "item_parent_language"
	TableLogEvents                      = "log_events"
	TableLogEventsUser                  = "log_events_user"
	TableOfDay                          = "of_day"
	TablePersonalMessages               = "personal_messages"
	TablePerspectives                   = "perspectives"
	TablePerspectivesGroups             = "perspectives_groups"
	TablePerspectivesGroupsPerspectives = "perspectives_groups_perspectives"
	TablePerspectivesPages              = "perspectives_pages"
	TablePicture                        = "pictures"
	TablePictureItem                    = "picture_item"
	TableSpec                           = "spec"
	TableTextstorageText                = "textstorage_text"
	UserTableName                       = "users"
	TableUserUserPreferences            = "user_user_preferences"
	TableVehicleVehicleType             = "vehicle_vehicle_type"
	TableVoting                         = "voting"
)

var (
	CommentMessageTable                      = goqu.T(CommentMessageTableName)
	CommentMessageTableColID                 = CommentMessageTable.Col("id")
	CommentMessageTableColTypeID             = CommentMessageTable.Col("type_id")
	CommentMessageTableColItemID             = CommentMessageTable.Col("item_id")
	CommentMessageTableColAuthorID           = CommentMessageTable.Col("author_id")
	CommentMessageTableColDatetime           = CommentMessageTable.Col("datetime")
	CommentMessageTableColVote               = CommentMessageTable.Col("vote")
	CommentMessageTableColParentID           = CommentMessageTable.Col("parent_id")
	CommentMessageTableColMessage            = CommentMessageTable.Col("message")
	CommentMessageTableColIP                 = CommentMessageTable.Col("ip")
	CommentMessageTableColDeleted            = CommentMessageTable.Col(CommentMessageTableDeletedColName)
	CommentMessageTableColModeratorAttention = CommentMessageTable.Col(CommentMessageTableModeratorAttentionColName)

	CommentTopicTable              = goqu.T(CommentTopicTableName)
	CommentTopicTableColItemID     = CommentTopicTable.Col("item_id")
	CommentTopicTableColTypeID     = CommentTopicTable.Col("type_id")
	CommentTopicTableColLastUpdate = CommentTopicTable.Col("last_update")

	CommentTopicSubscribeTable          = goqu.T(CommentTopicSubscribeTableName)
	CommentTopicSubscribeTableColItemID = CommentTopicSubscribeTable.Col("item_id")
	CommentTopicSubscribeTableColTypeID = CommentTopicSubscribeTable.Col("type_id")
	CommentTopicSubscribeTableColUserID = CommentTopicSubscribeTable.Col("user_id")

	CommentVoteTable = goqu.T(CommentVoteTableName)

	DfHashTable = goqu.T(DfHashTableName)

	ForumsThemesTable      = goqu.T(ForumsThemesTableName)
	ForumsThemesTableColID = ForumsThemesTable.Col("id")

	ForumsThemeParentTable                = goqu.T(ForumsThemeParentTableName)
	ForumsThemeParentTableColParentID     = ForumsThemeParentTable.Col("parent_id")
	ForumsThemeParentTableColForumThemeID = ForumsThemeParentTable.Col("forum_theme_id")

	ForumsTopicsTable               = goqu.T(ForumsTopicsTableName)
	ForumsTopicsTableColID          = ForumsTopicsTable.Col("id")
	ForumsTopicsTableColStatus      = ForumsTopicsTable.Col("status")
	ForumsTopicsTableColThemeID     = ForumsTopicsTable.Col("theme_id")
	ForumsTopicsTableColName        = ForumsTopicsTable.Col("name")
	ForumsTopicsTableColAddDatetime = ForumsTopicsTable.Col("add_datetime")
	ForumsTopicsTableColAuthorID    = ForumsTopicsTable.Col("author_id")

	ItemTable      = goqu.T(ItemTableName)
	ItemTableColID = ItemTable.Col("id")

	UserTable        = goqu.T(UserTableName)
	UserTableColID   = UserTable.Col("id")
	UserTableColRole = UserTable.Col("role")
)
