CREATE TABLE `image` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `filepath` varchar(255) NOT NULL,
  `filesize` int(10) unsigned NOT NULL,
  `width` int(10) unsigned NOT NULL,
  `height` int(10) unsigned NOT NULL,
  `date_add` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `dir` varchar(255) NOT NULL,
  `crop_left` smallint(5) unsigned NOT NULL DEFAULT '0',
  `crop_top` smallint(5) unsigned NOT NULL DEFAULT '0',
  `crop_width` smallint(5) unsigned NOT NULL DEFAULT '0',
  `crop_height` smallint(5) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `filename` (`filepath`,`dir`),
  KEY `image_dir_id` (`dir`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;