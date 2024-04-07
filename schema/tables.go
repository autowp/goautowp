package schema

import "github.com/doug-martin/goqu/v9"

const (
	ArticlesTableName     = "articles"
	TableAttrsAttributes  = "attrs_attributes"
	TableAttrsListOptions = "attrs_list_options"
	TableAttrsTypes       = "attrs_types"
	TableAttrsUnits       = "attrs_units"

	AttrsUserValuesTableName          = "attrs_user_values"
	AttrsUserValuesTableUserIDColName = "user_id"

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
	CommentTopicViewTableName           = "comment_topic_view"
	CommentVoteTableName                = "comment_vote"
	TableContact                        = "contact"
	TableDfDistance                     = "df_distance"
	DfHashTableName                     = "df_hash"
	TableFormattedImage                 = "formated_image"
	ForumsThemesTableName               = "forums_themes"
	ForumsThemeParentTableName          = "forums_theme_parent"
	ForumsTopicsTableName               = "forums_topics"
	HtmlsTableName                      = "htmls"
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

	UserTableName                     = "users"
	UserTableIDColName                = "id"
	UserTableSpecsVolumeColName       = "specs_volume"
	UserTableSpecsVolumeValidColName  = "specs_volume_valid"
	UserTableVotesLeftColName         = "votes_left"
	UserTableVotesPerDayColName       = "votes_per_day"
	UserTableLanguageColName          = "language"
	UserTableRoleColName              = "role"
	UserTableDeletedColName           = "deleted"
	UserTableUUIDColName              = "uuid"
	UserTableEmailColName             = "e_mail"
	UserTableEmailToCheckColName      = "email_to_check"
	UserTableHideEmailColName         = "hide_e_mail"
	UserTablePasswordColName          = "password"
	UserTableEmailCheckCodeColName    = "email_check_code"
	UserTableLastOnlineColName        = "last_online"
	UserTableTimezoneColName          = "timezone"
	UserTableLastIPColName            = "last_ip"
	UserTableRegDateColName           = "reg_date"
	UserTableLastMessageTimeColName   = "last_message_time"
	UserTableMessagingIntervalColName = "messaging_interval"
	UserTableIdentityColName          = "identity"
	UserTableNameColName              = "name"
	UserTableSpecsWeightColName       = "specs_weight"
	UserTableLoginColName             = "login"
	UserTableForumsMessagesColName    = "forums_messages"
	UserTableForumsTopicsColName      = "forums_topics"

	TableUserUserPreferences = "user_user_preferences"
	TableVehicleVehicleType  = "vehicle_vehicle_type"
	VotingTableName          = "voting"
)

var (
	ArticlesTable                   = goqu.T(ArticlesTableName)
	ArticlesTableIDCol              = ArticlesTable.Col("id")
	ArticlesTableNameCol            = ArticlesTable.Col("name")
	ArticlesTableCatnameCol         = ArticlesTable.Col("catname")
	ArticlesTableAuthorIDCol        = ArticlesTable.Col("author_id")
	ArticlesTableEnabledCol         = ArticlesTable.Col("enabled")
	ArticlesTableAddDateCol         = ArticlesTable.Col("add_date")
	ArticlesTablePreviewFilenameCol = ArticlesTable.Col("preview_filename")
	ArticlesTableDescriptionCol     = ArticlesTable.Col("description")
	ArticlesTableHTMLIDCol          = ArticlesTable.Col("html_id")

	AttrsUserValuesTable          = goqu.T(AttrsUserValuesTableName)
	AttrsUserValuesTableUserIDCol = AttrsUserValuesTable.Col(AttrsUserValuesTableUserIDColName)

	CommentMessageTable                      = goqu.T(CommentMessageTableName)
	CommentMessageTableIDCol                 = CommentMessageTable.Col("id")
	CommentMessageTableTypeIDCol             = CommentMessageTable.Col("type_id")
	CommentMessageTableItemIDCol             = CommentMessageTable.Col("item_id")
	CommentMessageTableAuthorIDCol           = CommentMessageTable.Col("author_id")
	CommentMessageTableDatetimeCol           = CommentMessageTable.Col("datetime")
	CommentMessageTableVoteCol               = CommentMessageTable.Col("vote")
	CommentMessageTableParentIDCol           = CommentMessageTable.Col("parent_id")
	CommentMessageTableMessageCol            = CommentMessageTable.Col("message")
	CommentMessageTableIPCol                 = CommentMessageTable.Col("ip")
	CommentMessageTableDeletedCol            = CommentMessageTable.Col(CommentMessageTableDeletedColName)
	CommentMessageTableModeratorAttentionCol = CommentMessageTable.Col(CommentMessageTableModeratorAttentionColName)

	CommentTopicTable              = goqu.T(CommentTopicTableName)
	CommentTopicTableItemIDCol     = CommentTopicTable.Col("item_id")
	CommentTopicTableTypeIDCol     = CommentTopicTable.Col("type_id")
	CommentTopicTableLastUpdateCol = CommentTopicTable.Col("last_update")

	CommentTopicViewTable             = goqu.T(CommentTopicViewTableName)
	CommentTopicViewTableUserIDCol    = CommentTopicViewTable.Col("user_id")
	CommentTopicViewTableTypeIDCol    = CommentTopicViewTable.Col("type_id")
	CommentTopicViewTableItemIDCol    = CommentTopicViewTable.Col("item_id")
	CommentTopicViewTableTimestampCol = CommentTopicViewTable.Col("timestamp")

	CommentTopicSubscribeTable          = goqu.T(CommentTopicSubscribeTableName)
	CommentTopicSubscribeTableItemIDCol = CommentTopicSubscribeTable.Col("item_id")
	CommentTopicSubscribeTableTypeIDCol = CommentTopicSubscribeTable.Col("type_id")
	CommentTopicSubscribeTableUserIDCol = CommentTopicSubscribeTable.Col("user_id")

	CommentVoteTable = goqu.T(CommentVoteTableName)

	DfHashTable = goqu.T(DfHashTableName)

	ForumsThemesTable      = goqu.T(ForumsThemesTableName)
	ForumsThemesTableIDCol = ForumsThemesTable.Col("id")

	ForumsThemeParentTable                = goqu.T(ForumsThemeParentTableName)
	ForumsThemeParentTableParentIDCol     = ForumsThemeParentTable.Col("parent_id")
	ForumsThemeParentTableForumThemeIDCol = ForumsThemeParentTable.Col("forum_theme_id")

	ForumsTopicsTable               = goqu.T(ForumsTopicsTableName)
	ForumsTopicsTableIDCol          = ForumsTopicsTable.Col("id")
	ForumsTopicsTableStatusCol      = ForumsTopicsTable.Col("status")
	ForumsTopicsTableThemeIDCol     = ForumsTopicsTable.Col("theme_id")
	ForumsTopicsTableNameCol        = ForumsTopicsTable.Col("name")
	ForumsTopicsTableAddDatetimeCol = ForumsTopicsTable.Col("add_datetime")
	ForumsTopicsTableAuthorIDCol    = ForumsTopicsTable.Col("author_id")

	ItemTable      = goqu.T(ItemTableName)
	ItemTableIDCol = ItemTable.Col("id")

	HtmlsTable        = goqu.T(HtmlsTableName)
	HtmlsTableIDCol   = HtmlsTable.Col("id")
	HtmlsTableHTMLCol = HtmlsTable.Col("html")

	UserTable                     = goqu.T(UserTableName)
	UserTableIDCol                = UserTable.Col(UserTableIDColName)
	UserTableRoleCol              = UserTable.Col(UserTableRoleColName)
	UserTableDeletedCol           = UserTable.Col(UserTableDeletedColName)
	UserTableNameCol              = UserTable.Col(UserTableNameColName)
	UserTableIdentityCol          = UserTable.Col(UserTableIdentityColName)
	UserTableLanguageCol          = UserTable.Col(UserTableLanguageColName)
	UserTablePicturesTotalCol     = UserTable.Col("pictures_total")
	UserTableSpecsVolumeCol       = UserTable.Col(UserTableSpecsVolumeColName)
	UserTableSpecsVolumeValidCol  = UserTable.Col(UserTableSpecsVolumeValidColName)
	UserTableVotesLeftCol         = UserTable.Col(UserTableVotesLeftColName)
	UserTableVotesPerDayCol       = UserTable.Col(UserTableVotesPerDayColName)
	UserTableUUIDCol              = UserTable.Col(UserTableUUIDColName)
	UserTableLastOnlineCol        = UserTable.Col(UserTableLastOnlineColName)
	UserTableSpecsWeightCol       = UserTable.Col(UserTableSpecsWeightColName)
	UserTableImgCol               = UserTable.Col("img")
	UserTableEMailCol             = UserTable.Col(UserTableEmailColName)
	UserTableEMailToCheckCol      = UserTable.Col(UserTableEmailToCheckColName)
	UserTableRegDateCol           = UserTable.Col(UserTableRegDateColName)
	UserTableLastMessageTimeCol   = UserTable.Col(UserTableLastMessageTimeColName)
	UserTableMessagingIntervalCol = UserTable.Col(UserTableMessagingIntervalColName)

	VotingTable      = goqu.T(VotingTableName)
	VotingTableIDCol = VotingTable.Col("id")
)
