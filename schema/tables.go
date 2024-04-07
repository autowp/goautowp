package schema

import "github.com/doug-martin/goqu/v9"

const (
	ArticlesTableName         = "articles"
	AttrsAttributesTableName  = "attrs_attributes"
	AttrsListOptionsTableName = "attrs_list_options"
	AttrsTypesTableName       = "attrs_types"
	AttrsUnitsTableName       = "attrs_units"

	AttrsUserValuesTableName          = "attrs_user_values"
	AttrsUserValuesTableUserIDColName = "user_id"

	TableAttrsValues             = "attrs_values"
	AttrsZonesTableName          = "attrs_zones"
	AttrsZoneAttributesTableName = "attrs_zone_attributes"
	CarTypesTableName            = "car_types"

	CommentMessageTableName                      = "comment_message"
	CommentMessageTableModeratorAttentionColName = "moderator_attention"
	CommentMessageTableDeletedColName            = "deleted"
	CommentMessageTableDeleteDateColName         = "delete_date"
	CommentMessageTableRepliesCountColName       = "replies_count"

	CommentTopicTableName          = "comment_topic"
	CommentTopicSubscribeTableName = "comment_topic_subscribe"
	CommentTopicViewTableName      = "comment_topic_view"
	CommentVoteTableName           = "comment_vote"
	TableContact                   = "contact"
	TableDfDistance                = "df_distance"
	DfHashTableName                = "df_hash"
	TableFormattedImage            = "formated_image"
	ForumsThemesTableName          = "forums_themes"
	ForumsThemeParentTableName     = "forums_theme_parent"
	ForumsTopicsTableName          = "forums_topics"
	HtmlsTableName                 = "htmls"
	TableImage                     = "image"

	ItemTableName                          = "item"
	ItemTableNameColName                   = "name"
	ItemTableCatnameColName                = "catname"
	ItemTableEngineItemIDColName           = "engine_item_id"
	ItemTableItemTypeIDColName             = "item_type_id"
	ItemTableIsConceptColName              = "is_concept"
	ItemTableIsConceptInheritColName       = "is_concept_inherit"
	ItemTableSpecIDColName                 = "spec_id"
	ItemTableIDColName                     = "id"
	ItemTableFullNameColName               = "full_name"
	ItemTableLogoIDColName                 = "logo_id"
	ItemTableBeginYearColName              = "begin_year"
	ItemTableEndYearColName                = "end_year"
	ItemTableBeginMonthColName             = "begin_month"
	ItemTableEndMonthColName               = "end_month"
	ItemTableBeginModelYearColName         = "begin_model_year"
	ItemTableEndModelYearColName           = "end_model_year"
	ItemTableBeginModelYearFractionColName = "begin_model_year_fraction"
	ItemTableEndModelYearFractionColName   = "end_model_year_fraction"
	ItemTableTodayColName                  = "today"
	ItemTableBodyColName                   = "body"
	ItemTableIsGroupColName                = "is_group"
	ItemTableProducedExactlyColName        = "produced_exactly"

	TableItemParentCache                    = "item_parent_cache"
	TableItemLanguage                       = "item_language"
	TableItemParent                         = "item_parent"
	TableItemParentLanguage                 = "item_parent_language"
	TableLogEvents                          = "log_events"
	TableLogEventsUser                      = "log_events_user"
	TableOfDay                              = "of_day"
	TablePersonalMessages                   = "personal_messages"
	PerspectivesTableName                   = "perspectives"
	PerspectivesGroupsTableName             = "perspectives_groups"
	PerspectivesGroupsPerspectivesTableName = "perspectives_groups_perspectives"
	PerspectivesPagesTableName              = "perspectives_pages"

	PictureTableName            = "pictures"
	PictureTableIDColName       = "id"
	PictureTableImageIDColName  = "image_id"
	PictureTableIdentityColName = "identity"
	PictureTableIPColName       = "ip"
	PictureTableOwnerIDColName  = "owner_id"
	PictureTableStatusColName   = "status"

	TablePictureItem = "picture_item"
	TableSpec        = "spec"

	TextstorageTextTableName      = "textstorage_text"
	TextstorageTextTableIDColName = "id"

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

	TableUserUserPreferences    = "user_user_preferences"
	VehicleVehicleTypeTableName = "vehicle_vehicle_type"
	VotingTableName             = "voting"
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

	AttrsAttributesTable               = goqu.T(AttrsAttributesTableName)
	AttrsAttributesTableIDCol          = AttrsAttributesTable.Col("id")
	AttrsAttributesTableNameCol        = AttrsAttributesTable.Col("name")
	AttrsAttributesTableDescriptionCol = AttrsAttributesTable.Col("description")
	AttrsAttributesTableTypeIDCol      = AttrsAttributesTable.Col("type_id")
	AttrsAttributesTableUnitIDCol      = AttrsAttributesTable.Col("unit_id")
	AttrsAttributesTableMultipleCol    = AttrsAttributesTable.Col("multiple")
	AttrsAttributesTablePrecisionCol   = AttrsAttributesTable.Col("precision")
	AttrsAttributesTableParentIDCol    = AttrsAttributesTable.Col("parent_id")
	AttrsAttributesTablePositionCol    = AttrsAttributesTable.Col("position")

	AttrsListOptionsTable               = goqu.T(AttrsListOptionsTableName)
	AttrsListOptionsTableIDCol          = AttrsListOptionsTable.Col("id")
	AttrsListOptionsTableNameCol        = AttrsListOptionsTable.Col("name")
	AttrsListOptionsTableAttributeIDCol = AttrsListOptionsTable.Col("attribute_id")
	AttrsListOptionsTableParentIDCol    = AttrsListOptionsTable.Col("parent_id")
	AttrsListOptionsTablePositionCol    = AttrsListOptionsTable.Col("position")

	AttrsTypesTable        = goqu.T(AttrsTypesTableName)
	AttrsTypesTableIDCol   = AttrsTypesTable.Col("id")
	AttrsTypesTableNameCol = AttrsTypesTable.Col("name")

	AttrsUnitsTable        = goqu.T(AttrsUnitsTableName)
	AttrsUnitsTableIDCol   = AttrsUnitsTable.Col("id")
	AttrsUnitsTableNameCol = AttrsUnitsTable.Col("name")
	AttrsUnitsTableAbbrCol = AttrsUnitsTable.Col("abbr")

	AttrsUserValuesTable          = goqu.T(AttrsUserValuesTableName)
	AttrsUserValuesTableUserIDCol = AttrsUserValuesTable.Col(AttrsUserValuesTableUserIDColName)

	AttrsZoneAttributesTable               = goqu.T(AttrsZoneAttributesTableName)
	AttrsZoneAttributesTableZoneIDCol      = AttrsZoneAttributesTable.Col("zone_id")
	AttrsZoneAttributesTableAttributeIDCol = AttrsZoneAttributesTable.Col("attribute_id")
	AttrsZoneAttributesTablePositionCol    = AttrsZoneAttributesTable.Col("position")

	AttrsZonesTable        = goqu.T(AttrsZonesTableName)
	AttrsZonesTableIDCol   = AttrsZonesTable.Col("id")
	AttrsZonesTableNameCol = AttrsZonesTable.Col("name")

	CarTypesTable            = goqu.T(CarTypesTableName)
	CarTypesTableIDCol       = CarTypesTable.Col("id")
	CarTypesTableNameCol     = CarTypesTable.Col("name")
	CarTypesTableCatnameCol  = CarTypesTable.Col("catname")
	CarTypesTablePositionCol = CarTypesTable.Col("position")
	CarTypesTableParentIDCol = CarTypesTable.Col("parent_id")

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
	CommentTopicTableMessagesCol   = CommentTopicTable.Col("messages")

	CommentTopicViewTable             = goqu.T(CommentTopicViewTableName)
	CommentTopicViewTableUserIDCol    = CommentTopicViewTable.Col("user_id")
	CommentTopicViewTableTypeIDCol    = CommentTopicViewTable.Col("type_id")
	CommentTopicViewTableItemIDCol    = CommentTopicViewTable.Col("item_id")
	CommentTopicViewTableTimestampCol = CommentTopicViewTable.Col("timestamp")

	CommentTopicSubscribeTable          = goqu.T(CommentTopicSubscribeTableName)
	CommentTopicSubscribeTableItemIDCol = CommentTopicSubscribeTable.Col("item_id")
	CommentTopicSubscribeTableTypeIDCol = CommentTopicSubscribeTable.Col("type_id")
	CommentTopicSubscribeTableUserIDCol = CommentTopicSubscribeTable.Col("user_id")

	CommentVoteTable             = goqu.T(CommentVoteTableName)
	CommentVoteTableUserIDCol    = CommentVoteTable.Col("user_id")
	CommentVoteTableCommentIDCol = CommentVoteTable.Col("comment_id")
	CommentVoteTableVoteCol      = CommentVoteTable.Col("vote")

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

	ItemTable                  = goqu.T(ItemTableName)
	ItemTableIDCol             = ItemTable.Col(ItemTableIDColName)
	ItemTableNameCol           = ItemTable.Col(ItemTableNameColName)
	ItemTableBeginYearCol      = ItemTable.Col(ItemTableBeginYearColName)
	ItemTableBeginModelYearCol = ItemTable.Col(ItemTableBeginModelYearColName)
	ItemTableIsGroupCol        = ItemTable.Col(ItemTableIsGroupColName)
	ItemTableItemTypeIDCol     = ItemTable.Col(ItemTableItemTypeIDColName)

	HtmlsTable        = goqu.T(HtmlsTableName)
	HtmlsTableIDCol   = HtmlsTable.Col("id")
	HtmlsTableHTMLCol = HtmlsTable.Col("html")

	PerspectivesTable            = goqu.T(PerspectivesTableName)
	PerspectivesTableIDCol       = PerspectivesTable.Col("id")
	PerspectivesTablePositionCol = PerspectivesTable.Col("position")
	PerspectivesTableNameCol     = PerspectivesTable.Col("name")

	PerspectivesGroupsTable            = goqu.T(PerspectivesGroupsTableName)
	PerspectivesGroupsTableIDCol       = PerspectivesGroupsTable.Col("id")
	PerspectivesGroupsTableNameCol     = PerspectivesGroupsTable.Col("name")
	PerspectivesGroupsTablePageIDCol   = PerspectivesGroupsTable.Col("page_id")
	PerspectivesGroupsTablePositionCol = PerspectivesGroupsTable.Col("position")

	PerspectivesGroupsPerspectivesTable                 = goqu.T(PerspectivesGroupsPerspectivesTableName)
	PerspectivesGroupsPerspectivesTablePerspectiveIDCol = PerspectivesGroupsPerspectivesTable.Col("perspective_id")
	PerspectivesGroupsPerspectivesTableGroupIDCol       = PerspectivesGroupsPerspectivesTable.Col("group_id")
	PerspectivesGroupsPerspectivesTablePositionCol      = PerspectivesGroupsPerspectivesTable.Col("position")

	PerspectivesPagesTable        = goqu.T(PerspectivesPagesTableName)
	PerspectivesPagesTableIDCol   = PerspectivesPagesTable.Col("id")
	PerspectivesPagesTableNameCol = PerspectivesPagesTable.Col("name")

	PictureTable            = goqu.T(PictureTableName)
	PictureTableIDCol       = PictureTable.Col(PictureTableIDColName)
	PictureTableIdentityCol = PictureTable.Col(PictureTableIdentityColName)
	PictureTableOwnerIDCol  = PictureTable.Col(PictureTableOwnerIDColName)
	PictureTableStatusCol   = PictureTable.Col(PictureTableStatusColName)

	TextstorageTextTable        = goqu.T(TextstorageTextTableName)
	TextstorageTextTableIDCol   = TextstorageTextTable.Col(TextstorageTextTableIDColName)
	TextstorageTextTableTextCol = TextstorageTextTable.Col("text")

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

	VehicleVehicleTypeTable                 = goqu.T(VehicleVehicleTypeTableName)
	VehicleVehicleTypeTableVehicleTypeIDCol = VehicleVehicleTypeTable.Col("vehicle_type_id")
	VehicleVehicleTypeTableVehicleIDCol     = VehicleVehicleTypeTable.Col("vehicle_id")
	VehicleVehicleTypeTableInheritedCol     = VehicleVehicleTypeTable.Col("inherited")

	VotingTable      = goqu.T(VotingTableName)
	VotingTableIDCol = VotingTable.Col("id")
)
