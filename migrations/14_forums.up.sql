CREATE TABLE `forums_themes` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` int(10) unsigned DEFAULT NULL,
  `folder` varchar(30) NOT NULL DEFAULT '',
  `name` varchar(50) NOT NULL DEFAULT '',
  `position` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `description` tinytext NOT NULL,
  `topics` int(10) unsigned NOT NULL DEFAULT '0',
  `messages` int(10) unsigned NOT NULL DEFAULT '0',
  `is_moderator` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `disable_topics` tinyint(4) unsigned DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `folder` (`folder`),
  UNIQUE KEY `caption` (`name`),
  KEY `parent_id` (`parent_id`),
  CONSTRAINT `FK_forums_themes_forums_themes_id` FOREIGN KEY (`parent_id`) REFERENCES `forums_themes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `forums_theme_parent` (
  `forum_theme_id` int(11) unsigned NOT NULL,
  `parent_id` int(11) unsigned NOT NULL,
  `level` tinyint(4) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`forum_theme_id`,`parent_id`),
  KEY `FK_forum_theme_parent_forums_themes_id2` (`parent_id`),
  CONSTRAINT `FK_forum_theme_parent_forums_themes_id` FOREIGN KEY (`forum_theme_id`) REFERENCES `forums_themes` (`id`),
  CONSTRAINT `FK_forum_theme_parent_forums_themes_id2` FOREIGN KEY (`parent_id`) REFERENCES `forums_themes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `forums_theme_parent` VALUES (2,2,0),(3,3,0),(4,4,1),(4,5,0),(5,5,0),(6,6,1),(6,16,0),(7,7,1),(7,16,0),(8,8,1),
(8,16,0),(9,9,1),(9,16,0),(10,10,1),(10,16,0),(11,11,1),(11,16,0),(12,12,1),(12,16,0),(13,13,1),(13,16,0),(14,14,1),
(14,16,0),(15,15,0),(16,16,0);

CREATE TABLE `forums_topics` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `theme_id` int(11) unsigned DEFAULT '0',
  `name` varchar(100) NOT NULL DEFAULT '',
  `author_id` int(10) unsigned NOT NULL DEFAULT '0',
  `add_datetime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `_messages` int(10) unsigned NOT NULL DEFAULT '0',
  `views` int(10) unsigned NOT NULL DEFAULT '0',
  `status` enum('normal','closed','deleted') NOT NULL DEFAULT 'normal',
  `author_ip` varbinary(16) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `theme_id` (`theme_id`),
  KEY `author_id` (`author_id`),
  CONSTRAINT `forums_topics_fk` FOREIGN KEY (`theme_id`) REFERENCES `forums_themes` (`id`),
  CONSTRAINT `forums_topics_fk1` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `forums_topics_subscribers` (
  `topic_id` int(10) unsigned NOT NULL,
  `user_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`topic_id`,`user_id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `topics_subscribers_fk` FOREIGN KEY (`topic_id`) REFERENCES `forums_topics` (`id`) ON DELETE CASCADE,
  CONSTRAINT `topics_subscribers_fk1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;