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

	AttrsValuesTableName = "attrs_values"

	AttrsZonesTableName          = "attrs_zones"
	AttrsZoneAttributesTableName = "attrs_zone_attributes"

	BrandAliasTableName          = "brand_alias"
	BrandAliasTableItemIDColName = "item_id"
	BrandAliasTableNameColName   = "name"

	CarTypesTableName = "car_types"

	CommentMessageTableName                      = "comment_message"
	CommentMessageTableIDColName                 = "id"
	CommentMessageTableParentIDColName           = "parent_id"
	CommentMessageTableTypeIDColName             = "type_id"
	CommentMessageTableItemIDColName             = "item_id"
	CommentMessageTableAuthorIDColName           = "author_id"
	CommentMessageTableModeratorAttentionColName = "moderator_attention"
	CommentMessageTableDeletedColName            = "deleted"
	CommentMessageTableDeletedByColName          = "deleted_by"
	CommentMessageTableDeleteDateColName         = "delete_date"
	CommentMessageTableRepliesCountColName       = "replies_count"
	CommentMessageTableVoteColName               = "vote"

	CommentTopicTableName              = "comment_topic"
	CommentTopicTableItemIDColName     = "item_id"
	CommentTopicTableTypeIDColName     = "type_id"
	CommentTopicTableLastUpdateColName = "last_update"
	CommentTopicTableMessagesColName   = "messages"

	CommentTopicSubscribeTableName          = "comment_topic_subscribe"
	CommentTopicSubscribeTableItemIDColName = "item_id"
	CommentTopicSubscribeTableTypeIDColName = "type_id"
	CommentTopicSubscribeTableUserIDColName = "user_id"
	CommentTopicSubscribeTableSentColName   = "sent"

	CommentTopicViewTableName             = "comment_topic_view"
	CommentTopicViewTableUserIDColName    = "user_id"
	CommentTopicViewTableTypeIDColName    = "type_id"
	CommentTopicViewTableItemIDColName    = "item_id"
	CommentTopicViewTableTimestampColName = "timestamp"

	CommentVoteTableName             = "comment_vote"
	CommentVoteTableUserIDColName    = "user_id"
	CommentVoteTableCommentIDColName = "comment_id"
	CommentVoteTableVoteColName      = "vote"

	ContactTableName                 = "contact"
	ContactTableUserIDColName        = "user_id"
	ContactTableContactUserIDColName = "contact_user_id"
	ContactTableTimestampColName     = "timestamp"

	DfDistanceTableName                = "df_distance"
	DfDistanceTableDistanceColName     = "distance"
	DfDistanceTableSrcPictureIDColName = "src_picture_id"
	DfDistanceTableDstPictureIDColName = "dst_picture_id"
	DfDistanceTableHideColName         = "hide"

	DfHashTableName             = "df_hash"
	DfHashTableHashColName      = "hash"
	DfHashTablePictureIDColName = "picture_id"

	FormattedImageTableName                    = "formated_image"
	FormattedImageTableStatusColName           = "status"
	FormattedImageTableImageIDColName          = "image_id"
	FormattedImageTableFormatColName           = "format"
	FormattedImageTableFormattedImageIDColName = "formated_image_id"

	ForumsThemesTableName            = "forums_themes"
	ForumsThemesTableTopicsColName   = "topics"
	ForumsThemesTableMessagesColName = "messages"

	ForumsThemeParentTableName = "forums_theme_parent"

	ForumsTopicsTableName               = "forums_topics"
	ForumsTopicsTableIDColName          = "id"
	ForumsTopicsTableStatusColName      = "status"
	ForumsTopicsTableThemeIDColName     = "theme_id"
	ForumsTopicsTableNameColName        = "name"
	ForumsTopicsTableAddDatetimeColName = "add_datetime"
	ForumsTopicsTableAuthorIDColName    = "author_id"
	ForumsTopicsTableAuthorIPColName    = "author_ip"
	ForumsTopicsTableViewsColName       = "views"

	HtmlsTableName = "htmls"

	ImageDirTableName         = "image_dir"
	ImageDirTableCountColName = "count"
	ImageDirTableDirColName   = "dir"

	ImageTableName              = "image"
	ImageTableIDColName         = "id"
	ImageTableFilepathColName   = "filepath"
	ImageTableFilesizeColName   = "filesize"
	ImageTableWidthColName      = "width"
	ImageTableHeightColName     = "height"
	ImageTableDirColName        = "dir"
	ImageTableDateAddColName    = "date_add"
	ImageTableCropLeftColName   = "crop_left"
	ImageTableCropTopColName    = "crop_top"
	ImageTableCropWidthColName  = "crop_width"
	ImageTableCropHeightColName = "crop_height"
	ImageTableS3ColName         = "s3"

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
	ItemTableAddDatetimeColName            = "add_datetime"

	ItemPointTableName          = "item_point"
	ItemPointTableItemIDColName = "item_id"
	ItemPointTablePointColName  = "point"

	ItemParentCacheTableName            = "item_parent_cache"
	ItemParentCacheTableItemIDColName   = "item_id"
	ItemParentCacheTableParentIDColName = "parent_id"
	ItemParentCacheTableDiffColName     = "diff"
	ItemParentCacheTableTuningColName   = "tuning"
	ItemParentCacheTableSportColName    = "sport"
	ItemParentCacheTableDesignColName   = "design"

	ItemLanguageTableName              = "item_language"
	ItemLanguageTableItemIDColName     = "item_id"
	ItemLanguageTableLanguageColName   = "language"
	ItemLanguageTableNameColName       = "name"
	ItemLanguageTableTextIDColName     = "text_id"
	ItemLanguageTableFullTextIDColName = "full_text_id"

	ItemParentTableName             = "item_parent"
	ItemParentTableParentIDColName  = "parent_id"
	ItemParentTableItemIDColName    = "item_id"
	ItemParentTableTypeColName      = "type"
	ItemParentTableCatnameColName   = "catname"
	ItemParentTableTimestampColName = "timestamp"

	ItemParentLanguageTableName            = "item_parent_language"
	ItemParentLanguageTableItemIDColName   = "item_id"
	ItemParentLanguageTableParentIDColName = "parent_id"
	ItemParentLanguageTableLanguageColName = "language"
	ItemParentLanguageTableNameColName     = "name"
	ItemParentLanguageTableIsAutoColName   = "is_auto"

	LinksTableName          = "links"
	LinksTableIDColName     = "id"
	LinksTableNameColName   = "name"
	LinksTableURLColName    = "url"
	LinksTableTypeColName   = "type"
	LinksTableItemIDColName = "item_id"

	LogEventsTableName               = "log_events"
	LogEventsTableIDColName          = "id"
	LogEventsTableDescriptionColName = "description"
	LogEventsTableUserIDColName      = "user_id"
	LogEventsTableAddDatetimeColName = "add_datetime"

	LogEventsArticlesTableName              = "log_events_articles"
	LogEventsArticlesTableLogEventIDColName = "log_event_id"
	LogEventsArticlesTableArticleIDColName  = "article_id"

	LogEventsItemTableName              = "log_events_item"
	LogEventsItemTableLogEventIDColName = "log_event_id"
	LogEventsItemTableItemIDColName     = "item_id"

	LogEventsPicturesTableName              = "log_events_pictures"
	LogEventsPicturesTableLogEventIDColName = "log_event_id"
	LogEventsPicturesTablePictureIDColName  = "picture_id"

	LogEventsUserTableName              = "log_events_user"
	LogEventsUserTableLogEventIDColName = "log_event_id"
	LogEventsUserTableUserIDColName     = "user_id"

	OfDayTableName           = "of_day"
	OfDayTableItemIDColName  = "item_id"
	OfDayTableUserIDColName  = "user_id"
	OfDayTableDayDateColName = "day_date"

	PersonalMessagesTableName                 = "personal_messages"
	PersonalMessagesTableAddDatetimeColName   = "add_datetime"
	PersonalMessagesTableContentsColName      = "contents"
	PersonalMessagesTableDeletedByFromColName = "deleted_by_from"
	PersonalMessagesTableDeletedByToColName   = "deleted_by_to"
	PersonalMessagesTableFromUserIDColName    = "from_user_id"
	PersonalMessagesTableToUserIDColName      = "to_user_id"
	PersonalMessagesTableReadenColName        = "readen"

	PerspectivesTableName                   = "perspectives"
	PerspectivesGroupsTableName             = "perspectives_groups"
	PerspectivesGroupsPerspectivesTableName = "perspectives_groups_perspectives"
	PerspectivesPagesTableName              = "perspectives_pages"

	PicturesModerVotesTableName             = "pictures_moder_votes"
	PicturesModerVotesTableUserIDColName    = "user_id"
	PicturesModerVotesTablePictureIDColName = "picture_id"
	PicturesModerVotesTableVoteColName      = "vote"
	PicturesModerVotesTableReasonColName    = "reason"
	PicturesModerVotesTableDayDateColName   = "day_date"

	PictureModerVoteTemplateTableName          = "picture_moder_vote_template"
	PictureModerVoteTemplateTableIDColName     = "id"
	PictureModerVoteTemplateTableReasonColName = "reason"
	PictureModerVoteTemplateTableVoteColName   = "vote"
	PictureModerVoteTemplateTableUserIDColName = "user_id"

	PictureTableName                      = "pictures"
	PictureTableIDColName                 = "id"
	PictureTableImageIDColName            = "image_id"
	PictureTableIdentityColName           = "identity"
	PictureTableIPColName                 = "ip"
	PictureTableOwnerIDColName            = "owner_id"
	PictureTableStatusColName             = "status"
	PictureTableChangeStatusUserIDColName = "change_status_user_id"

	PictureItemTableName             = "picture_item"
	PictureItemTablePictureIDColName = "picture_id"
	PictureItemTableItemIDColName    = "item_id"

	PictureViewTableName             = "picture_view"
	PictureViewTablePictureIDColName = "picture_id"
	PictureViewTableViewsColName     = "views"

	PictureVoteTableName             = "picture_vote"
	PictureVoteTablePictureIDColName = "picture_id"
	PictureVoteTableUserIDColName    = "user_id"
	PictureVoteTableValueColName     = "value"
	PictureVoteTableTimestampColName = "timestamp"

	PictureVoteSummaryTableName             = "picture_vote_summary"
	PictureVoteSummaryTablePictureIDColName = "picture_id"
	PictureVoteSummaryTablePositiveColName  = "positive"
	PictureVoteSummaryTableNegativeColName  = "negative"

	SpecTableName             = "spec"
	SpecTableIDColName        = "id"
	SpecTableNameColName      = "name"
	SpecTableShortNameColName = "short_name"
	SpecTableParentIDColName  = "parent_id"

	TelegramBrandTableName = "telegram_brand"

	TelegramChatTableName = "telegram_chat"

	TextstorageRevisionTableName             = "textstorage_revision"
	TextstorageRevisionTableTextIDColName    = "text_id"
	TextstorageRevisionTableRevisionColName  = "revision"
	TextstorageRevisionTableTextColName      = "text"
	TextstorageRevisionTableTimestampColName = "timestamp"
	TextstorageRevisionTableUserIDColName    = "user_id"

	TextstorageTextTableName               = "textstorage_text"
	TextstorageTextTableIDColName          = "id"
	TextstorageTextTableTextColName        = "text"
	TextstorageTextTableLastUpdatedColName = "last_updated"
	TextstorageTextTableRevisionColName    = "revision"

	UserAccountTableName = "user_account"

	UserTableName                     = "users"
	UserTableIDColName                = "id"
	UserTableSpecsVolumeColName       = "specs_volume"
	UserTableSpecsVolumeValidColName  = "specs_volume_valid"
	UserTableVotesLeftColName         = "votes_left"
	UserTableVotesPerDayColName       = "votes_per_day"
	UserTableLanguageColName          = "language"
	UserTablePicturesTotalColName     = "pictures_total"
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
	UserTablePicturesAddedColName     = "pictures_added"

	VehicleVehicleTypeTableName                 = "vehicle_vehicle_type"
	VehicleVehicleTypeTableVehicleTypeIDColName = "vehicle_type_id"
	VehicleVehicleTypeTableVehicleIDColName     = "vehicle_id"
	VehicleVehicleTypeTableInheritedColName     = "inherited"

	VotingTableName = "voting"
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
	AttrsUserValuesTableItemIDCol = AttrsUserValuesTable.Col("item_id")
	AttrsUserValuesTableWeightCol = AttrsUserValuesTable.Col("weight")

	AttrsValuesTable = goqu.T(AttrsValuesTableName)

	AttrsZoneAttributesTable               = goqu.T(AttrsZoneAttributesTableName)
	AttrsZoneAttributesTableZoneIDCol      = AttrsZoneAttributesTable.Col("zone_id")
	AttrsZoneAttributesTableAttributeIDCol = AttrsZoneAttributesTable.Col("attribute_id")
	AttrsZoneAttributesTablePositionCol    = AttrsZoneAttributesTable.Col("position")

	AttrsZonesTable        = goqu.T(AttrsZonesTableName)
	AttrsZonesTableIDCol   = AttrsZonesTable.Col("id")
	AttrsZonesTableNameCol = AttrsZonesTable.Col("name")

	BrandAliasTable          = goqu.T(BrandAliasTableName)
	BrandAliasTableItemIDCol = BrandAliasTable.Col(BrandAliasTableItemIDColName)
	BrandAliasTableNameCol   = BrandAliasTable.Col(BrandAliasTableNameColName)

	CarTypesTable            = goqu.T(CarTypesTableName)
	CarTypesTableIDCol       = CarTypesTable.Col("id")
	CarTypesTableNameCol     = CarTypesTable.Col("name")
	CarTypesTableCatnameCol  = CarTypesTable.Col("catname")
	CarTypesTablePositionCol = CarTypesTable.Col("position")
	CarTypesTableParentIDCol = CarTypesTable.Col("parent_id")

	CommentMessageTable                      = goqu.T(CommentMessageTableName)
	CommentMessageTableIDCol                 = CommentMessageTable.Col(CommentMessageTableIDColName)
	CommentMessageTableTypeIDCol             = CommentMessageTable.Col(CommentMessageTableTypeIDColName)
	CommentMessageTableItemIDCol             = CommentMessageTable.Col(CommentMessageTableItemIDColName)
	CommentMessageTableAuthorIDCol           = CommentMessageTable.Col(CommentMessageTableAuthorIDColName)
	CommentMessageTableDatetimeCol           = CommentMessageTable.Col("datetime")
	CommentMessageTableVoteCol               = CommentMessageTable.Col(CommentMessageTableVoteColName)
	CommentMessageTableParentIDCol           = CommentMessageTable.Col(CommentMessageTableParentIDColName)
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
	CommentTopicViewTableUserIDCol    = CommentTopicViewTable.Col(CommentTopicViewTableUserIDColName)
	CommentTopicViewTableTypeIDCol    = CommentTopicViewTable.Col(CommentTopicViewTableTypeIDColName)
	CommentTopicViewTableItemIDCol    = CommentTopicViewTable.Col(CommentTopicViewTableItemIDColName)
	CommentTopicViewTableTimestampCol = CommentTopicViewTable.Col(CommentTopicViewTableTimestampColName)

	CommentTopicSubscribeTable          = goqu.T(CommentTopicSubscribeTableName)
	CommentTopicSubscribeTableItemIDCol = CommentTopicSubscribeTable.Col(CommentTopicSubscribeTableItemIDColName)
	CommentTopicSubscribeTableTypeIDCol = CommentTopicSubscribeTable.Col(CommentTopicSubscribeTableTypeIDColName)
	CommentTopicSubscribeTableUserIDCol = CommentTopicSubscribeTable.Col(CommentTopicSubscribeTableUserIDColName)

	CommentVoteTable             = goqu.T(CommentVoteTableName)
	CommentVoteTableUserIDCol    = CommentVoteTable.Col(CommentVoteTableUserIDColName)
	CommentVoteTableCommentIDCol = CommentVoteTable.Col(CommentVoteTableCommentIDColName)
	CommentVoteTableVoteCol      = CommentVoteTable.Col(CommentVoteTableVoteColName)

	ContactTable                 = goqu.T(ContactTableName)
	ContactTableUserIDCol        = ContactTable.Col("user_id")
	ContactTableContactUserIDCol = ContactTable.Col("contact_user_id")
	ContactTableTimestampCol     = ContactTable.Col("timestamp")

	DfDistanceTable                = goqu.T(DfDistanceTableName)
	DfDistanceTableDistanceCol     = DfDistanceTable.Col(DfDistanceTableDistanceColName)
	DfDistanceTableSrcPictureIDCol = DfDistanceTable.Col(DfDistanceTableSrcPictureIDColName)
	DfDistanceTableDstPictureIDCol = DfDistanceTable.Col(DfDistanceTableDstPictureIDColName)

	DfHashTable             = goqu.T(DfHashTableName)
	DfHashTableHashCol      = DfHashTable.Col(DfHashTableHashColName)
	DfHashTablePictureIDCol = DfHashTable.Col(DfHashTablePictureIDColName)

	FormattedImageTable                    = goqu.T(FormattedImageTableName)
	FormattedImageTableStatusCol           = FormattedImageTable.Col(FormattedImageTableStatusColName)
	FormattedImageTableImageIDCol          = FormattedImageTable.Col(FormattedImageTableImageIDColName)
	FormattedImageTableFormatCol           = FormattedImageTable.Col(FormattedImageTableFormatColName)
	FormattedImageTableFormattedImageIDCol = FormattedImageTable.Col(FormattedImageTableFormattedImageIDColName)

	ForumsThemesTable                 = goqu.T(ForumsThemesTableName)
	ForumsThemesTableIDCol            = ForumsThemesTable.Col("id")
	ForumsThemesTableNameCol          = ForumsThemesTable.Col("name")
	ForumsThemesTableTopicsCol        = ForumsThemesTable.Col(ForumsThemesTableTopicsColName)
	ForumsThemesTableMessagesCol      = ForumsThemesTable.Col(ForumsThemesTableMessagesColName)
	ForumsThemesTableDisableTopicsCol = ForumsThemesTable.Col("disable_topics")
	ForumsThemesTableDescriptionCol   = ForumsThemesTable.Col("description")
	ForumsThemesTableIsModeratorCol   = ForumsThemesTable.Col("is_moderator")
	ForumsThemesTablePositionCol      = ForumsThemesTable.Col("position")
	ForumsThemesTableParentIDCol      = ForumsThemesTable.Col("parent_id")

	ForumsThemeParentTable                = goqu.T(ForumsThemeParentTableName)
	ForumsThemeParentTableParentIDCol     = ForumsThemeParentTable.Col("parent_id")
	ForumsThemeParentTableForumThemeIDCol = ForumsThemeParentTable.Col("forum_theme_id")

	ForumsTopicsTable               = goqu.T(ForumsTopicsTableName)
	ForumsTopicsTableIDCol          = ForumsTopicsTable.Col(ForumsTopicsTableIDColName)
	ForumsTopicsTableStatusCol      = ForumsTopicsTable.Col(ForumsTopicsTableStatusColName)
	ForumsTopicsTableThemeIDCol     = ForumsTopicsTable.Col(ForumsTopicsTableThemeIDColName)
	ForumsTopicsTableNameCol        = ForumsTopicsTable.Col(ForumsTopicsTableNameColName)
	ForumsTopicsTableAddDatetimeCol = ForumsTopicsTable.Col(ForumsTopicsTableAddDatetimeColName)
	ForumsTopicsTableAuthorIDCol    = ForumsTopicsTable.Col(ForumsTopicsTableAuthorIDColName)
	ForumsTopicsTableAuthorIPCol    = ForumsTopicsTable.Col(ForumsTopicsTableAuthorIPColName)
	ForumsTopicsTableViewsCol       = ForumsTopicsTable.Col(ForumsTopicsTableViewsColName)

	ImageTable              = goqu.T(ImageTableName)
	ImageTableIDCol         = ImageTable.Col(ImageTableIDColName)
	ImageTableWidthCol      = ImageTable.Col(ImageTableWidthColName)
	ImageTableHeightCol     = ImageTable.Col(ImageTableHeightColName)
	ImageTableFilesizeCol   = ImageTable.Col(ImageTableFilesizeColName)
	ImageTableFilepathCol   = ImageTable.Col(ImageTableFilepathColName)
	ImageTableDirCol        = ImageTable.Col(ImageTableDirColName)
	ImageTableCropLeftCol   = ImageTable.Col(ImageTableCropLeftColName)
	ImageTableCropTopCol    = ImageTable.Col(ImageTableCropTopColName)
	ImageTableCropWidthCol  = ImageTable.Col(ImageTableCropWidthColName)
	ImageTableCropHeightCol = ImageTable.Col(ImageTableCropHeightColName)

	ImageDirTable         = goqu.T(ImageDirTableName)
	ImageDirTableCountCol = ImageDirTable.Col(ImageDirTableCountColName)
	ImageDirTableDirCol   = ImageDirTable.Col(ImageDirTableDirColName)

	ItemLanguageTable              = goqu.T(ItemLanguageTableName)
	ItemLanguageTableItemIDCol     = ItemLanguageTable.Col(ItemLanguageTableItemIDColName)
	ItemLanguageTableLanguageCol   = ItemLanguageTable.Col(ItemLanguageTableLanguageColName)
	ItemLanguageTableNameCol       = ItemLanguageTable.Col(ItemLanguageTableNameColName)
	ItemLanguageTableTextIDCol     = ItemLanguageTable.Col(ItemLanguageTableTextIDColName)
	ItemLanguageTableFullTextIDCol = ItemLanguageTable.Col(ItemLanguageTableFullTextIDColName)

	ItemParentTable            = goqu.T(ItemParentTableName)
	ItemParentTableParentIDCol = ItemParentTable.Col(ItemParentTableParentIDColName)
	ItemParentTableItemIDCol   = ItemParentTable.Col(ItemParentTableItemIDColName)
	ItemParentTableTypeCol     = ItemParentTable.Col(ItemParentTableTypeColName)
	ItemParentTableCatnameCol  = ItemParentTable.Col(ItemParentTableCatnameColName)

	ItemParentCacheTable            = goqu.T(ItemParentCacheTableName)
	ItemParentCacheTableItemIDCol   = ItemParentCacheTable.Col(ItemParentCacheTableItemIDColName)
	ItemParentCacheTableParentIDCol = ItemParentCacheTable.Col(ItemParentCacheTableParentIDColName)

	ItemParentLanguageTable            = goqu.T(ItemParentLanguageTableName)
	ItemParentLanguageTableItemIDCol   = ItemParentLanguageTable.Col(ItemParentLanguageTableItemIDColName)
	ItemParentLanguageTableParentIDCol = ItemParentLanguageTable.Col(ItemParentLanguageTableParentIDColName)
	ItemParentLanguageTableLanguageCol = ItemParentLanguageTable.Col(ItemParentLanguageTableLanguageColName)
	ItemParentLanguageTableNameCol     = ItemParentLanguageTable.Col(ItemParentLanguageTableNameColName)

	ItemPointTable          = goqu.T(ItemPointTableName)
	ItemPointTablePointCol  = ItemPointTable.Col(ItemPointTablePointColName)
	ItemPointTableItemIDCol = ItemPointTable.Col(ItemPointTableItemIDColName)

	ItemTable                  = goqu.T(ItemTableName)
	ItemTableIDCol             = ItemTable.Col(ItemTableIDColName)
	ItemTableBodyCol           = ItemTable.Col(ItemTableBodyColName)
	ItemTableSpecIDCol         = ItemTable.Col(ItemTableSpecIDColName)
	ItemTableCatnameCol        = ItemTable.Col(ItemTableCatnameColName)
	ItemTableNameCol           = ItemTable.Col(ItemTableNameColName)
	ItemTableBeginYearCol      = ItemTable.Col(ItemTableBeginYearColName)
	ItemTableEndYearCol        = ItemTable.Col(ItemTableEndYearColName)
	ItemTableBeginModelYearCol = ItemTable.Col(ItemTableBeginModelYearColName)
	ItemTableEndModelYearCol   = ItemTable.Col(ItemTableEndModelYearColName)
	ItemTableIsGroupCol        = ItemTable.Col(ItemTableIsGroupColName)
	ItemTableItemTypeIDCol     = ItemTable.Col(ItemTableItemTypeIDColName)
	ItemTableTodayCol          = ItemTable.Col(ItemTableTodayColName)

	LinksTable          = goqu.T(LinksTableName)
	LinksTableIDCol     = LinksTable.Col("id")
	LinksTableNameCol   = LinksTable.Col("name")
	LinksTableURLCol    = LinksTable.Col("url")
	LinksTableTypeCol   = LinksTable.Col("type")
	LinksTableItemIDCol = LinksTable.Col("item_id")

	HtmlsTable        = goqu.T(HtmlsTableName)
	HtmlsTableIDCol   = HtmlsTable.Col("id")
	HtmlsTableHTMLCol = HtmlsTable.Col("html")

	LogEventsTable               = goqu.T(LogEventsTableName)
	LogEventsTableIDCol          = LogEventsTable.Col(LogEventsTableIDColName)
	LogEventsTableDescriptionCol = LogEventsTable.Col(LogEventsTableDescriptionColName)
	LogEventsTableUserIDCol      = LogEventsTable.Col(LogEventsTableUserIDColName)
	LogEventsTableAddDatetimeCol = LogEventsTable.Col(LogEventsTableAddDatetimeColName)

	LogEventsArticlesTable              = goqu.T(LogEventsArticlesTableName)
	LogEventsArticlesTableLogEventIDCol = LogEventsArticlesTable.Col(LogEventsArticlesTableLogEventIDColName)
	LogEventsArticlesTableArticleIDCol  = LogEventsArticlesTable.Col(LogEventsArticlesTableArticleIDColName)

	LogEventsItemTable              = goqu.T(LogEventsItemTableName)
	LogEventsItemTableLogEventIDCol = LogEventsItemTable.Col(LogEventsItemTableLogEventIDColName)
	LogEventsItemTableItemIDCol     = LogEventsItemTable.Col(LogEventsItemTableItemIDColName)

	LogEventsPicturesTable              = goqu.T(LogEventsPicturesTableName)
	LogEventsPicturesTableLogEventIDCol = LogEventsPicturesTable.Col(LogEventsPicturesTableLogEventIDColName)
	LogEventsPicturesTablePictureIDCol  = LogEventsPicturesTable.Col(LogEventsPicturesTablePictureIDColName)

	LogEventsUserTable = goqu.T(LogEventsUserTableName)

	OfDayTable           = goqu.T(OfDayTableName)
	OfDayTableDayDateCol = OfDayTable.Col(OfDayTableDayDateColName)
	OfDayTableItemIDCol  = OfDayTable.Col(OfDayTableItemIDColName)

	PersonalMessagesTable                 = goqu.T(PersonalMessagesTableName)
	PersonalMessagesTableIDCol            = PersonalMessagesTable.Col("id")
	PersonalMessagesTableAddDatetimeCol   = PersonalMessagesTable.Col(PersonalMessagesTableAddDatetimeColName)
	PersonalMessagesTableDeletedByFromCol = PersonalMessagesTable.Col(PersonalMessagesTableDeletedByFromColName)
	PersonalMessagesTableDeletedByToCol   = PersonalMessagesTable.Col(PersonalMessagesTableDeletedByToColName)
	PersonalMessagesTableFromUserIDCol    = PersonalMessagesTable.Col(PersonalMessagesTableFromUserIDColName)
	PersonalMessagesTableToUserIDCol      = PersonalMessagesTable.Col(PersonalMessagesTableToUserIDColName)
	PersonalMessagesTableReadenCol        = PersonalMessagesTable.Col(PersonalMessagesTableReadenColName)

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

	PictureModerVoteTemplateTable          = goqu.T(PictureModerVoteTemplateTableName)
	PictureModerVoteTemplateTableIDCol     = PictureModerVoteTemplateTable.Col(PictureModerVoteTemplateTableIDColName)
	PictureModerVoteTemplateTableReasonCol = PictureModerVoteTemplateTable.Col(PictureModerVoteTemplateTableReasonColName)
	PictureModerVoteTemplateTableVoteCol   = PictureModerVoteTemplateTable.Col(PictureModerVoteTemplateTableVoteColName)
	PictureModerVoteTemplateTableUserIDCol = PictureModerVoteTemplateTable.Col(PictureModerVoteTemplateTableUserIDColName)

	PicturesModerVotesTable             = goqu.T(PicturesModerVotesTableName)
	PicturesModerVotesTableUserIDCol    = PicturesModerVotesTable.Col(PicturesModerVotesTableUserIDColName)
	PicturesModerVotesTablePictureIDCol = PicturesModerVotesTable.Col(PicturesModerVotesTablePictureIDColName)

	PictureTable                      = goqu.T(PictureTableName)
	PictureTableIDCol                 = PictureTable.Col(PictureTableIDColName)
	PictureTableIdentityCol           = PictureTable.Col(PictureTableIdentityColName)
	PictureTableOwnerIDCol            = PictureTable.Col(PictureTableOwnerIDColName)
	PictureTableStatusCol             = PictureTable.Col(PictureTableStatusColName)
	PictureTableImageIDCol            = PictureTable.Col(PictureTableImageIDColName)
	PictureTableChangeStatusUserIDCol = PictureTable.Col(PictureTableChangeStatusUserIDColName)

	PictureItemTable             = goqu.T(PictureItemTableName)
	PictureItemTablePictureIDCol = PictureItemTable.Col(PictureItemTablePictureIDColName)
	PictureItemTableItemIDCol    = PictureItemTable.Col(PictureItemTableItemIDColName)

	PictureViewTable             = goqu.T(PictureViewTableName)
	PictureViewTablePictureIDCol = PictureViewTable.Col(PictureViewTablePictureIDColName)
	PictureViewTableViewsCol     = PictureViewTable.Col(PictureViewTableViewsColName)

	PictureVoteTable             = goqu.T(PictureVoteTableName)
	PictureVoteTablePictureIDCol = PictureVoteTable.Col(PictureVoteTablePictureIDColName)
	PictureVoteTableUserIDCol    = PictureVoteTable.Col(PictureVoteTableUserIDColName)
	PictureVoteTableValueCol     = PictureVoteTable.Col(PictureVoteTableValueColName)

	PictureVoteSummaryTable             = goqu.T(PictureVoteSummaryTableName)
	PictureVoteSummaryTablePictureIDCol = PictureVoteSummaryTable.Col(PictureVoteSummaryTablePictureIDColName)
	PictureVoteSummaryTablePositiveCol  = PictureVoteSummaryTable.Col(PictureVoteSummaryTablePositiveColName)
	PictureVoteSummaryTableNegativeCol  = PictureVoteSummaryTable.Col(PictureVoteSummaryTableNegativeColName)

	SpecTable             = goqu.T(SpecTableName)
	SpecTableIDCol        = SpecTable.Col(SpecTableIDColName)
	SpecTableNameCol      = SpecTable.Col(SpecTableNameColName)
	SpecTableShortNameCol = SpecTable.Col(SpecTableShortNameColName)
	SpecTableParentIDCol  = SpecTable.Col(SpecTableParentIDColName)

	TelegramBrandTable          = goqu.T(TelegramBrandTableName)
	TelegramBrandTableChatIDCol = TelegramBrandTable.Col("chat_id")

	TelegramChatTable            = goqu.T(TelegramChatTableName)
	TelegramChatTableChatIDCol   = TelegramChatTable.Col("chat_id")
	TelegramChatTableUserIDCol   = TelegramChatTable.Col("user_id")
	TelegramChatTableMessagesCol = TelegramChatTable.Col("messages")

	TextstorageRevisionTable             = goqu.T(TextstorageRevisionTableName)
	TextstorageRevisionTableTextIDCol    = TextstorageRevisionTable.Col(TextstorageRevisionTableTextIDColName)
	TextstorageRevisionTableRevisionCol  = TextstorageRevisionTable.Col(TextstorageRevisionTableRevisionColName)
	TextstorageRevisionTableTextCol      = TextstorageRevisionTable.Col(TextstorageRevisionTableTextColName)
	TextstorageRevisionTableTimestampCol = TextstorageRevisionTable.Col(TextstorageRevisionTableTimestampColName)
	TextstorageRevisionTableUserIDCol    = TextstorageRevisionTable.Col(TextstorageRevisionTableUserIDColName)

	TextstorageTextTable            = goqu.T(TextstorageTextTableName)
	TextstorageTextTableIDCol       = TextstorageTextTable.Col(TextstorageTextTableIDColName)
	TextstorageTextTableTextCol     = TextstorageTextTable.Col(TextstorageTextTableTextColName)
	TextstorageTextTableRevisionCol = TextstorageTextTable.Col(TextstorageTextTableRevisionColName)

	UserAccountTable             = goqu.T(UserAccountTableName)
	UserAccountTableUserIDCol    = UserAccountTable.Col("user_id")
	UserAccountTableServiceIDCol = UserAccountTable.Col("service_id")

	UserItemSubscribeTable          = goqu.T("user_item_subscribe")
	UserItemSubscribeTableUserIDCol = UserItemSubscribeTable.Col("user_id")

	UserTable                     = goqu.T(UserTableName)
	UserTableIDCol                = UserTable.Col(UserTableIDColName)
	UserTableRoleCol              = UserTable.Col(UserTableRoleColName)
	UserTableDeletedCol           = UserTable.Col(UserTableDeletedColName)
	UserTableNameCol              = UserTable.Col(UserTableNameColName)
	UserTableIdentityCol          = UserTable.Col(UserTableIdentityColName)
	UserTableLanguageCol          = UserTable.Col(UserTableLanguageColName)
	UserTablePicturesTotalCol     = UserTable.Col(UserTablePicturesTotalColName)
	UserTableSpecsVolumeCol       = UserTable.Col(UserTableSpecsVolumeColName)
	UserTableSpecsVolumeValidCol  = UserTable.Col(UserTableSpecsVolumeValidColName)
	UserTableVotesLeftCol         = UserTable.Col(UserTableVotesLeftColName)
	UserTableVotesPerDayCol       = UserTable.Col(UserTableVotesPerDayColName)
	UserTableUUIDCol              = UserTable.Col(UserTableUUIDColName)
	UserTableLastOnlineCol        = UserTable.Col(UserTableLastOnlineColName)
	UserTableLastIPCol            = UserTable.Col(UserTableLastIPColName)
	UserTableSpecsWeightCol       = UserTable.Col(UserTableSpecsWeightColName)
	UserTableImgCol               = UserTable.Col("img")
	UserTableEmailCol             = UserTable.Col(UserTableEmailColName)
	UserTableEmailToCheckCol      = UserTable.Col(UserTableEmailToCheckColName)
	UserTableRegDateCol           = UserTable.Col(UserTableRegDateColName)
	UserTableLastMessageTimeCol   = UserTable.Col(UserTableLastMessageTimeColName)
	UserTableMessagingIntervalCol = UserTable.Col(UserTableMessagingIntervalColName)
	UserTableLoginCol             = UserTable.Col(UserTableLoginColName)
	UserTablePasswordCol          = UserTable.Col(UserTablePasswordColName)
	UserTableTimezoneCol          = UserTable.Col(UserTableTimezoneColName)
	UserTablePicturesAddedCol     = UserTable.Col(UserTablePicturesAddedColName)

	VehicleVehicleTypeTable                 = goqu.T(VehicleVehicleTypeTableName)
	VehicleVehicleTypeTableVehicleTypeIDCol = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableVehicleTypeIDColName)
	VehicleVehicleTypeTableVehicleIDCol     = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableVehicleIDColName)
	VehicleVehicleTypeTableInheritedCol     = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableInheritedColName)

	VotingTable      = goqu.T(VotingTableName)
	VotingTableIDCol = VotingTable.Col("id")
)
