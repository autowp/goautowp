CREATE TABLE `image` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `filepath` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `filename` (`filepath`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `pictures` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `image_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `image_id` (`image_id`),
  CONSTRAINT `pictures_fk7` FOREIGN KEY (`image_id`) REFERENCES `image` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `df_hash` (
  `picture_id` int(10) unsigned NOT NULL,
  `hash` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`picture_id`),
  CONSTRAINT `df_hash_ibfk_1` FOREIGN KEY (`picture_id`) REFERENCES `pictures` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `df_distance` (
  `src_picture_id` int(10) unsigned NOT NULL,
  `dst_picture_id` int(10) unsigned NOT NULL,
  `distance` tinyint(4) NOT NULL,
  `hide` tinyint(4) NOT NULL DEFAULT '0',
  PRIMARY KEY (`src_picture_id`,`dst_picture_id`),
  KEY `dst_picture_id` (`dst_picture_id`),
  CONSTRAINT `df_distance_ibfk_1` FOREIGN KEY (`src_picture_id`) REFERENCES `pictures` (`id`) ON DELETE CASCADE,
  CONSTRAINT `df_distance_ibfk_2` FOREIGN KEY (`dst_picture_id`) REFERENCES `pictures` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;