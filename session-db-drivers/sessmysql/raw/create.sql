CREATE TABLE IF NOT EXISTS `goboots_sessid` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `sid` varchar(40) NOT NULL DEFAULT '',
  `time` datetime NOT NULL,
  `updated` datetime NOT NULL,
  `expires` datetime NOT NULL,
  `data` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `sid` (`sid`),
  KEY `updated` (`updated`),
  KEY `expires` (`expires`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;